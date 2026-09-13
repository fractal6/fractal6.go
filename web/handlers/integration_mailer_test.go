//go:build integration

package handlers_test

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph"
	"fractale/fractal6.go/graph/codec"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/internal/testutil"
	"fractale/fractal6.go/internal/tools"
	"fractale/fractal6.go/web/handlers"
	"fractale/fractal6.go/web/sessions"
)

func TestMailingAttachments(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	publicKey, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	handlers.SetTestPostalKey(t, base64.StdEncoding.EncodeToString(publicKey))

	for _, role := range []model.RoleType{model.RoleTypeMember, model.RoleTypeGuest} {
		t.Run(string(role), func(t *testing.T) {
			org, username := "mailing-"+strings.ToLower(string(role)), testutil.TestUser
			author := &model.UserRef{Username: &username}
			guestCanCreate := false
			node := model.AddNodeInput{
				CreatedBy: author, CreatedAt: tools.Now(), Nameid: org, Rootnameid: org, Name: org,
				IsRoot: true, Type: model.NodeTypeCircle, Visibility: model.NodeVisibilityPublic,
				Mode: model.NodeModeCoordinated, GuestCanCreateTension: &guestCanCreate,
			}
			member := node
			member.Nameid, member.Name = codec.MemberIdCodec(org, username), string(role)
			member.Type, member.IsRoot = model.NodeTypeRole, false
			member.RoleType, member.FirstLink = &role, author
			memberRef := tools.StructMap[model.NodeRef](member)
			node.Children = []*model.NodeRef{&memberRef}
			if _, err := db.GetDB().Add(db.GetDB().GetRootUctx(), "node", node); err != nil {
				t.Fatalf("seed mailing org: %v", err)
			}
			t.Cleanup(func() {
				tids, err := db.GetDB().GetIDs("Tension.receiverid", org, nil, nil)
				if err != nil {
					t.Error(err)
				}
				for _, tid := range tids {
					if err := db.GetDB().DeleteTensionDeep(tid); err != nil {
						t.Error(err)
					}
				}
				cleanup := db.QueryMut{
					Q: `query {
						n as var(func: eq(Node.rootnameid, "{{.org}}"))
						u as var(func: eq(User.username, "{{.username}}"))
						a as var(func: eq(Activity.ownerid, "o#{{.org}}"))
					}`,
					M: []db.X{{D: `uid(u) <User.roles> uid(n) .
						uid(n) * * .
						uid(a) * * .`}},
				}
				if _, err := db.GetDB().Gamma(cleanup, map[string]string{"org": org, "username": username}); err != nil {
					t.Error(err)
				}
			})

			filename, image := org+".png", pngBytes()
			form := handlers.EmailForm{
				From: testutil.TestEmail, To: org + "@fractale.co", Title: org + " body",
				Msg: org + " ![image](cid:" + filename + ")", AttachmentQuantity: 1,
				Attachments: []handlers.InboundAttachment{{
					Filename: filename, ContentType: "image/png", Size: len(image),
					Data: base64.StdEncoding.EncodeToString(image),
				}},
			}
			payload, err := json.Marshal(form)
			if err != nil {
				t.Fatal(err)
			}
			digest := sha1.Sum(payload)
			signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA1, digest[:])
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodPost, "/mailing", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Postal-Signature", base64.StdEncoding.EncodeToString(signature))
			subscriber := sessions.GetCache().Subscribe(t.Context(), "api-tension-notification")
			defer subscriber.Close()
			if _, err := subscriber.ReceiveTimeout(t.Context(), 5*time.Second); err != nil {
				t.Fatal(err)
			}
			rr := httptest.NewRecorder()
			testRouter.ServeHTTP(rr, req)

			vars := map[string]string{"title": form.Title, "message": tools.QuoteString(form.Msg), "filename": filename}
			if role == model.RoleTypeGuest {
				requireStatus(t, rr, http.StatusBadRequest)
				if !strings.Contains(rr.Body.String(), "Guest cannot create tension") {
					t.Fatalf("expected post-insert authorization refusal, got %q", rr.Body.String())
				}
				q := db.QueryMut{Q: `{
					all(func: has(dgraph.type)) @filter(eq(Tension.title, "{{.title}}") OR
						eq(Post.message, "{{.message}}") OR eq(File.filename, "{{.filename}}")) { uid }
				}`}
				rows, err := db.Gamma[model.Comment](q, vars)
				if err != nil || len(rows) != 0 {
					t.Fatalf("rejected creation left tension/comment/file data: %+v, %v", rows, err)
				}
				return
			}

			requireStatus(t, rr, http.StatusOK)
			q := db.QueryMut{Q: `{
				all(func: type(Tension)) @filter(eq(Tension.title, "{{.title}}")) {
					uid Tension.comments {
						uid Post.message Post.createdBy { User.username }
						Comment.files { uid File.storageKey File.embedded File.comment { uid } File.tension { uid } }
					}
				}
			}`}
			tension, err := tools.First(db.Gamma[model.Tension](q, vars))
			if err != nil || tension.ID == "" || len(tension.Comments) != 1 || len(tension.Comments[0].Files) != 1 {
				t.Fatalf("expected one body comment with one file: %+v, %v", tension, err)
			}
			comment := tension.Comments[0]
			file := comment.Files[0]
			t.Cleanup(func() { purgeFile(t, file.ID, file.StorageKey) })
			if file.Comment == nil || file.Comment.ID != comment.ID || file.Tension == nil || file.Tension.ID != tension.ID {
				t.Fatalf("wrong attachment anchor: %+v", file)
			}
			want := strings.Replace(form.Msg, "cid:"+filename, "/file/"+file.ID, 1)
			if comment.CreatedBy == nil || comment.CreatedBy.Username != username || comment.Message != want || file.Embedded == nil || !*file.Embedded {
				t.Fatalf("body comment not preserved and rewritten: %+v", comment)
			}
			body, _, err := testStorageCli.GetObject(t.Context(), file.StorageKey)
			if err != nil {
				t.Fatal(err)
			}
			defer body.Close()
			got, err := io.ReadAll(body)
			if err != nil || !bytes.Equal(got, image) {
				t.Fatalf("attachment bytes differ: %v", err)
			}

			var notif model.EventNotif
			timeout := time.After(5 * time.Second)
			for notif.Tid != tension.ID {
				select {
				case message, ok := <-subscriber.Channel():
					if !ok {
						t.Fatal("notification subscription closed")
					}
					if err := json.Unmarshal([]byte(message.Payload), &notif); err != nil {
						t.Fatal(err)
					}
				case <-timeout:
					t.Fatal("mailing did not publish a notification")
				}
			}
			// Settle poll: the notifier waits until the comment's declared
			// attachments are anchored on it (one file here), never longer.
			interval := time.Second
			prevInterval := tools.ViperPositiveInt("notify.upload_poll_interval_ms", 1500)
			prevAttempts := tools.ViperPositiveInt("notify.upload_poll_attempts", 30)
			viper.Set("notify.upload_poll_interval_ms", 1000)
			viper.Set("notify.upload_poll_attempts", 2)
			t.Cleanup(func() {
				viper.Set("notify.upload_poll_interval_ms", prevInterval)
				viper.Set("notify.upload_poll_attempts", prevAttempts)
			})
			prevCap := tools.ViperPositiveInt("notify.inbound_attachment_max_count", 20)
			t.Cleanup(func() { viper.Set("notify.inbound_attachment_max_count", prevCap) })
			for _, tt := range []struct {
				name     string
				event    model.TensionEvent
				expected int
				cap_     int
				wait     bool
			}{
				{"inbound creation", model.TensionEventCreated, 0, prevCap, false},
				{"inbound reply", model.TensionEventCommentPushed, 0, prevCap, false},
				{"declared upload landed", model.TensionEventCreated, 1, prevCap, false},
				{"declared upload missing", model.TensionEventCommentPushed, 3, prevCap, true},
				// A lying client cannot wait longer than the cap allows.
				{"declared count capped", model.TensionEventCommentPushed, 999, 1, false},
			} {
				t.Run(tt.name, func(t *testing.T) {
					viper.Set("notify.inbound_attachment_max_count", tt.cap_)
					q := db.QueryMut{
						Q: `query { c as var(func: uid({{.cid}})) }`,
						M: []db.X{{S: `uid(c) <Comment.expected_attachments> "{{.n}}"^^<xs:int> .`}},
					}
					if _, err := db.GetDB().Gamma(q, map[string]string{"cid": comment.ID, "n": strconv.Itoa(tt.expected)}); err != nil {
						t.Fatal(err)
					}
					current := notif
					event := *notif.History[0]
					event.ID, event.EventType = nil, &tt.event
					current.History = []*model.EventRef{&event}
					start := time.Now()
					if err := graph.PushEventNotifications(current); err != nil {
						t.Fatal(err)
					}
					elapsed := time.Since(start)
					if tt.wait != (elapsed >= interval) {
						t.Errorf("notification took %v with expected_attachments=%d (poll interval %v)", elapsed, tt.expected, interval)
					}
				})
			}

			// Wait for background writes before deleting their fixtures.
			deadline := time.Now().Add(5 * time.Second)
			for {
				message, err := db.GetDB().GetByUid(tension.ID, "Post.message")
				if err != nil {
					t.Fatal(err)
				}
				traced, err := db.GetDB().Exists("Activity.ownerid", "o#"+org, nil)
				if err != nil {
					t.Fatal(err)
				}
				if message == want && traced {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("creation did not finish its search/activity writes")
				}
				time.Sleep(10 * time.Millisecond)
			}
		})
	}
}
