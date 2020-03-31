// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package model

import (
	"fmt"
	"io"
	"strconv"
)

type Node interface {
	IsNode()
}

type Post interface {
	IsPost()
}

type AddCircleInput struct {
	CreatedAt   string        `json:"createdAt"`
	CreatedBy   *UserRef      `json:"createdBy"`
	Parent      *NodeRef      `json:"parent"`
	Children    []*NodeRef    `json:"children"`
	Name        string        `json:"name"`
	Nameid      string        `json:"nameid"`
	Mandate     *MandateRef   `json:"mandate"`
	TensionsOut []*TensionRef `json:"tensions_out"`
	TensionsIn  []*TensionRef `json:"tensions_in"`
	IsRoot      bool          `json:"isRoot"`
}

type AddCirclePayload struct {
	Circle  []*Circle `json:"circle"`
	NumUids *int      `json:"numUids"`
}

type AddMandateInput struct {
	CreatedAt        string   `json:"createdAt"`
	Author           *UserRef `json:"author"`
	Message          *string  `json:"message"`
	Purpose          string   `json:"purpose"`
	Responsabilities *string  `json:"responsabilities"`
	Domains          []string `json:"domains"`
}

type AddMandatePayload struct {
	Mandate []*Mandate `json:"mandate"`
	NumUids *int       `json:"numUids"`
}

type AddRoleInput struct {
	CreatedAt   string        `json:"createdAt"`
	CreatedBy   *UserRef      `json:"createdBy"`
	Parent      *NodeRef      `json:"parent"`
	Children    []*NodeRef    `json:"children"`
	Name        string        `json:"name"`
	Nameid      string        `json:"nameid"`
	Mandate     *MandateRef   `json:"mandate"`
	TensionsOut []*TensionRef `json:"tensions_out"`
	TensionsIn  []*TensionRef `json:"tensions_in"`
	User        *UserRef      `json:"user"`
	Second      *UserRef      `json:"second"`
	Third       *UserRef      `json:"third"`
	Skills      []string      `json:"skills"`
}

type AddRolePayload struct {
	Role    []*Role `json:"role"`
	NumUids *int    `json:"numUids"`
}

type AddTensionInput struct {
	CreatedAt   string      `json:"createdAt"`
	Author      *UserRef    `json:"author"`
	Message     *string     `json:"message"`
	Nth         int         `json:"nth"`
	Title       string      `json:"title"`
	Type        TensionType `json:"type_"`
	Emitter     *NodeRef    `json:"emitter"`
	Receivers   []*NodeRef  `json:"receivers"`
	IsAnonymous bool        `json:"isAnonymous"`
	Severity    int         `json:"severity"`
	Comments    []*PostRef  `json:"comments"`
}

type AddTensionPayload struct {
	Tension []*Tension `json:"tension"`
	NumUids *int       `json:"numUids"`
}

type AddUserInput struct {
	CreatedAt string     `json:"createdAt"`
	Username  string     `json:"username"`
	Fullname  *string    `json:"fullname"`
	Password  string     `json:"password"`
	Roles     []*RoleRef `json:"roles"`
	Bio       *string    `json:"bio"`
}

type AddUserPayload struct {
	User    []*User `json:"user"`
	NumUids *int    `json:"numUids"`
}

type Circle struct {
	IsRoot      bool       `json:"isRoot"`
	ID          string     `json:"id"`
	CreatedAt   string     `json:"createdAt"`
	CreatedBy   *User      `json:"createdBy"`
	Parent      Node       `json:"parent"`
	Children    []Node     `json:"children"`
	Name        string     `json:"name"`
	Nameid      string     `json:"nameid"`
	Mandate     *Mandate   `json:"mandate"`
	TensionsOut []*Tension `json:"tensions_out"`
	TensionsIn  []*Tension `json:"tensions_in"`
}

func (Circle) IsNode() {}

type CircleFilter struct {
	ID        []string          `json:"id"`
	CreatedAt *DateTimeFilter   `json:"createdAt"`
	Name      *StringTermFilter `json:"name"`
	Nameid    *StringHashFilter `json:"nameid"`
	And       *CircleFilter     `json:"and"`
	Or        *CircleFilter     `json:"or"`
	Not       *CircleFilter     `json:"not"`
}

type CircleOrder struct {
	Asc  *CircleOrderable `json:"asc"`
	Desc *CircleOrderable `json:"desc"`
	Then *CircleOrder     `json:"then"`
}

type CirclePatch struct {
	CreatedAt   *string       `json:"createdAt"`
	CreatedBy   *UserRef      `json:"createdBy"`
	Parent      *NodeRef      `json:"parent"`
	Children    []*NodeRef    `json:"children"`
	Name        *string       `json:"name"`
	Mandate     *MandateRef   `json:"mandate"`
	TensionsOut []*TensionRef `json:"tensions_out"`
	TensionsIn  []*TensionRef `json:"tensions_in"`
	IsRoot      *bool         `json:"isRoot"`
}

type CircleRef struct {
	ID          *string       `json:"id"`
	CreatedAt   *string       `json:"createdAt"`
	CreatedBy   *UserRef      `json:"createdBy"`
	Parent      *NodeRef      `json:"parent"`
	Children    []*NodeRef    `json:"children"`
	Name        *string       `json:"name"`
	Nameid      *string       `json:"nameid"`
	Mandate     *MandateRef   `json:"mandate"`
	TensionsOut []*TensionRef `json:"tensions_out"`
	TensionsIn  []*TensionRef `json:"tensions_in"`
	IsRoot      *bool         `json:"isRoot"`
}

type DateTimeFilter struct {
	Eq *string `json:"eq"`
	Le *string `json:"le"`
	Lt *string `json:"lt"`
	Ge *string `json:"ge"`
	Gt *string `json:"gt"`
}

type DeleteCirclePayload struct {
	Msg     *string `json:"msg"`
	NumUids *int    `json:"numUids"`
}

type DeleteMandatePayload struct {
	Msg     *string `json:"msg"`
	NumUids *int    `json:"numUids"`
}

type DeleteNodePayload struct {
	Msg     *string `json:"msg"`
	NumUids *int    `json:"numUids"`
}

type DeletePostPayload struct {
	Msg     *string `json:"msg"`
	NumUids *int    `json:"numUids"`
}

type DeleteRolePayload struct {
	Msg     *string `json:"msg"`
	NumUids *int    `json:"numUids"`
}

type DeleteTensionPayload struct {
	Msg     *string `json:"msg"`
	NumUids *int    `json:"numUids"`
}

type DeleteUserPayload struct {
	Msg     *string `json:"msg"`
	NumUids *int    `json:"numUids"`
}

type FloatFilter struct {
	Eq *float64 `json:"eq"`
	Le *float64 `json:"le"`
	Lt *float64 `json:"lt"`
	Ge *float64 `json:"ge"`
	Gt *float64 `json:"gt"`
}

type IntFilter struct {
	Eq *int `json:"eq"`
	Le *int `json:"le"`
	Lt *int `json:"lt"`
	Ge *int `json:"ge"`
	Gt *int `json:"gt"`
}

type Mandate struct {
	Purpose          string   `json:"purpose"`
	Responsabilities *string  `json:"responsabilities"`
	Domains          []string `json:"domains"`
	ID               string   `json:"id"`
	CreatedAt        string   `json:"createdAt"`
	CreatedBy        *User    `json:"createdBy"`
	Message          *string  `json:"message"`
}

func (Mandate) IsPost() {}

type MandateFilter struct {
	ID        []string              `json:"id"`
	CreatedAt *DateTimeFilter       `json:"createdAt"`
	Message   *StringFullTextFilter `json:"message"`
	Purpose   *StringFullTextFilter `json:"purpose"`
	And       *MandateFilter        `json:"and"`
	Or        *MandateFilter        `json:"or"`
	Not       *MandateFilter        `json:"not"`
}

type MandateOrder struct {
	Asc  *MandateOrderable `json:"asc"`
	Desc *MandateOrderable `json:"desc"`
	Then *MandateOrder     `json:"then"`
}

type MandatePatch struct {
	CreatedAt        *string  `json:"createdAt"`
	Author           *UserRef `json:"author"`
	Message          *string  `json:"message"`
	Purpose          *string  `json:"purpose"`
	Responsabilities *string  `json:"responsabilities"`
	Domains          []string `json:"domains"`
}

type MandateRef struct {
	ID               *string  `json:"id"`
	CreatedAt        *string  `json:"createdAt"`
	Author           *UserRef `json:"author"`
	Message          *string  `json:"message"`
	Purpose          *string  `json:"purpose"`
	Responsabilities *string  `json:"responsabilities"`
	Domains          []string `json:"domains"`
}

type NodeFilter struct {
	ID        []string          `json:"id"`
	CreatedAt *DateTimeFilter   `json:"createdAt"`
	Name      *StringTermFilter `json:"name"`
	Nameid    *StringHashFilter `json:"nameid"`
	And       *NodeFilter       `json:"and"`
	Or        *NodeFilter       `json:"or"`
	Not       *NodeFilter       `json:"not"`
}

type NodeOrder struct {
	Asc  *NodeOrderable `json:"asc"`
	Desc *NodeOrderable `json:"desc"`
	Then *NodeOrder     `json:"then"`
}

type NodePatch struct {
	CreatedAt   *string       `json:"createdAt"`
	CreatedBy   *UserRef      `json:"createdBy"`
	Parent      *NodeRef      `json:"parent"`
	Children    []*NodeRef    `json:"children"`
	Name        *string       `json:"name"`
	Mandate     *MandateRef   `json:"mandate"`
	TensionsOut []*TensionRef `json:"tensions_out"`
	TensionsIn  []*TensionRef `json:"tensions_in"`
}

type NodeRef struct {
	ID     *string `json:"id"`
	Nameid *string `json:"nameid"`
}

type PostFilter struct {
	ID        []string              `json:"id"`
	CreatedAt *DateTimeFilter       `json:"createdAt"`
	Message   *StringFullTextFilter `json:"message"`
	And       *PostFilter           `json:"and"`
	Or        *PostFilter           `json:"or"`
	Not       *PostFilter           `json:"not"`
}

type PostOrder struct {
	Asc  *PostOrderable `json:"asc"`
	Desc *PostOrderable `json:"desc"`
	Then *PostOrder     `json:"then"`
}

type PostPatch struct {
	CreatedAt *string  `json:"createdAt"`
	Author    *UserRef `json:"author"`
	Message   *string  `json:"message"`
}

type PostRef struct {
	ID string `json:"id"`
}

type Role struct {
	User        *User      `json:"user"`
	Second      *User      `json:"second"`
	Skills      []string   `json:"skills"`
	ID          string     `json:"id"`
	CreatedAt   string     `json:"createdAt"`
	CreatedBy   *User      `json:"createdBy"`
	Parent      Node       `json:"parent"`
	Children    []Node     `json:"children"`
	Name        string     `json:"name"`
	Nameid      string     `json:"nameid"`
	Mandate     *Mandate   `json:"mandate"`
	TensionsOut []*Tension `json:"tensions_out"`
	TensionsIn  []*Tension `json:"tensions_in"`
}

func (Role) IsNode() {}

type RoleFilter struct {
	ID        []string          `json:"id"`
	CreatedAt *DateTimeFilter   `json:"createdAt"`
	Name      *StringTermFilter `json:"name"`
	Nameid    *StringHashFilter `json:"nameid"`
	Skills    *StringTermFilter `json:"skills"`
	And       *RoleFilter       `json:"and"`
	Or        *RoleFilter       `json:"or"`
	Not       *RoleFilter       `json:"not"`
}

type RoleOrder struct {
	Asc  *RoleOrderable `json:"asc"`
	Desc *RoleOrderable `json:"desc"`
	Then *RoleOrder     `json:"then"`
}

type RolePatch struct {
	CreatedAt   *string       `json:"createdAt"`
	CreatedBy   *UserRef      `json:"createdBy"`
	Parent      *NodeRef      `json:"parent"`
	Children    []*NodeRef    `json:"children"`
	Name        *string       `json:"name"`
	Mandate     *MandateRef   `json:"mandate"`
	TensionsOut []*TensionRef `json:"tensions_out"`
	TensionsIn  []*TensionRef `json:"tensions_in"`
	User        *UserRef      `json:"user"`
	Second      *UserRef      `json:"second"`
	Third       *UserRef      `json:"third"`
	Skills      []string      `json:"skills"`
}

type RoleRef struct {
	ID          *string       `json:"id"`
	CreatedAt   *string       `json:"createdAt"`
	CreatedBy   *UserRef      `json:"createdBy"`
	Parent      *NodeRef      `json:"parent"`
	Children    []*NodeRef    `json:"children"`
	Name        *string       `json:"name"`
	Nameid      *string       `json:"nameid"`
	Mandate     *MandateRef   `json:"mandate"`
	TensionsOut []*TensionRef `json:"tensions_out"`
	TensionsIn  []*TensionRef `json:"tensions_in"`
	User        *UserRef      `json:"user"`
	Second      *UserRef      `json:"second"`
	Third       *UserRef      `json:"third"`
	Skills      []string      `json:"skills"`
}

type StringExactFilter struct {
	Eq *string `json:"eq"`
	Le *string `json:"le"`
	Lt *string `json:"lt"`
	Ge *string `json:"ge"`
	Gt *string `json:"gt"`
}

type StringFullTextFilter struct {
	Alloftext *string `json:"alloftext"`
	Anyoftext *string `json:"anyoftext"`
}

type StringHashFilter struct {
	Eq *string `json:"eq"`
}

type StringRegExpFilter struct {
	Regexp *string `json:"regexp"`
}

type StringTermFilter struct {
	Allofterms *string `json:"allofterms"`
	Anyofterms *string `json:"anyofterms"`
}

type Tension struct {
	Nth         int         `json:"nth"`
	Title       string      `json:"title"`
	Type        TensionType `json:"type_"`
	Emitter     Node        `json:"emitter"`
	Receivers   []Node      `json:"receivers"`
	IsAnonymous bool        `json:"isAnonymous"`
	Severity    int         `json:"severity"`
	Comments    []Post      `json:"comments"`
	ID          string      `json:"id"`
	CreatedAt   string      `json:"createdAt"`
	CreatedBy   *User       `json:"createdBy"`
	Message     *string     `json:"message"`
}

func (Tension) IsPost() {}

type TensionFilter struct {
	ID        []string              `json:"id"`
	CreatedAt *DateTimeFilter       `json:"createdAt"`
	Message   *StringFullTextFilter `json:"message"`
	Title     *StringTermFilter     `json:"title"`
	Type      *TensionTypeHash      `json:"type_"`
	And       *TensionFilter        `json:"and"`
	Or        *TensionFilter        `json:"or"`
	Not       *TensionFilter        `json:"not"`
}

type TensionOrder struct {
	Asc  *TensionOrderable `json:"asc"`
	Desc *TensionOrderable `json:"desc"`
	Then *TensionOrder     `json:"then"`
}

type TensionPatch struct {
	CreatedAt   *string      `json:"createdAt"`
	Author      *UserRef     `json:"author"`
	Message     *string      `json:"message"`
	Nth         *int         `json:"nth"`
	Title       *string      `json:"title"`
	Type        *TensionType `json:"type_"`
	Emitter     *NodeRef     `json:"emitter"`
	Receivers   []*NodeRef   `json:"receivers"`
	IsAnonymous *bool        `json:"isAnonymous"`
	Severity    *int         `json:"severity"`
	Comments    []*PostRef   `json:"comments"`
}

type TensionRef struct {
	ID          *string      `json:"id"`
	CreatedAt   *string      `json:"createdAt"`
	Author      *UserRef     `json:"author"`
	Message     *string      `json:"message"`
	Nth         *int         `json:"nth"`
	Title       *string      `json:"title"`
	Type        *TensionType `json:"type_"`
	Emitter     *NodeRef     `json:"emitter"`
	Receivers   []*NodeRef   `json:"receivers"`
	IsAnonymous *bool        `json:"isAnonymous"`
	Severity    *int         `json:"severity"`
	Comments    []*PostRef   `json:"comments"`
}

type TensionTypeHash struct {
	Eq TensionType `json:"eq"`
}

type UpdateCircleInput struct {
	Filter *CircleFilter `json:"filter"`
	Set    *CirclePatch  `json:"set"`
	Remove *CirclePatch  `json:"remove"`
}

type UpdateCirclePayload struct {
	Circle  []*Circle `json:"circle"`
	NumUids *int      `json:"numUids"`
}

type UpdateMandateInput struct {
	Filter *MandateFilter `json:"filter"`
	Set    *MandatePatch  `json:"set"`
	Remove *MandatePatch  `json:"remove"`
}

type UpdateMandatePayload struct {
	Mandate []*Mandate `json:"mandate"`
	NumUids *int       `json:"numUids"`
}

type UpdateNodeInput struct {
	Filter *NodeFilter `json:"filter"`
	Set    *NodePatch  `json:"set"`
	Remove *NodePatch  `json:"remove"`
}

type UpdateNodePayload struct {
	Node    []Node `json:"node"`
	NumUids *int   `json:"numUids"`
}

type UpdatePostInput struct {
	Filter *PostFilter `json:"filter"`
	Set    *PostPatch  `json:"set"`
	Remove *PostPatch  `json:"remove"`
}

type UpdatePostPayload struct {
	Post    []Post `json:"post"`
	NumUids *int   `json:"numUids"`
}

type UpdateRoleInput struct {
	Filter *RoleFilter `json:"filter"`
	Set    *RolePatch  `json:"set"`
	Remove *RolePatch  `json:"remove"`
}

type UpdateRolePayload struct {
	Role    []*Role `json:"role"`
	NumUids *int    `json:"numUids"`
}

type UpdateTensionInput struct {
	Filter *TensionFilter `json:"filter"`
	Set    *TensionPatch  `json:"set"`
	Remove *TensionPatch  `json:"remove"`
}

type UpdateTensionPayload struct {
	Tension []*Tension `json:"tension"`
	NumUids *int       `json:"numUids"`
}

type UpdateUserInput struct {
	Filter *UserFilter `json:"filter"`
	Set    *UserPatch  `json:"set"`
	Remove *UserPatch  `json:"remove"`
}

type UpdateUserPayload struct {
	User    []*User `json:"user"`
	NumUids *int    `json:"numUids"`
}

type User struct {
	ID          string  `json:"id"`
	CreatedAt   string  `json:"createdAt"`
	Username    string  `json:"username"`
	Fullname    *string `json:"fullname"`
	Password    string  `json:"password"`
	Roles       []*Role `json:"roles"`
	BackedRoles []*Role `json:"backed_roles"`
	Bio         *string `json:"bio"`
}

type UserFilter struct {
	ID        []string          `json:"id"`
	CreatedAt *DateTimeFilter   `json:"createdAt"`
	Username  *StringHashFilter `json:"username"`
	And       *UserFilter       `json:"and"`
	Or        *UserFilter       `json:"or"`
	Not       *UserFilter       `json:"not"`
}

type UserOrder struct {
	Asc  *UserOrderable `json:"asc"`
	Desc *UserOrderable `json:"desc"`
	Then *UserOrder     `json:"then"`
}

type UserPatch struct {
	CreatedAt *string    `json:"createdAt"`
	Fullname  *string    `json:"fullname"`
	Password  *string    `json:"password"`
	Roles     []*RoleRef `json:"roles"`
	Bio       *string    `json:"bio"`
}

type UserRef struct {
	ID        *string    `json:"id"`
	CreatedAt *string    `json:"createdAt"`
	Username  *string    `json:"username"`
	Fullname  *string    `json:"fullname"`
	Password  *string    `json:"password"`
	Roles     []*RoleRef `json:"roles"`
	Bio       *string    `json:"bio"`
}

type CircleOrderable string

const (
	CircleOrderableCreatedAt CircleOrderable = "createdAt"
	CircleOrderableName      CircleOrderable = "name"
	CircleOrderableNameid    CircleOrderable = "nameid"
)

var AllCircleOrderable = []CircleOrderable{
	CircleOrderableCreatedAt,
	CircleOrderableName,
	CircleOrderableNameid,
}

func (e CircleOrderable) IsValid() bool {
	switch e {
	case CircleOrderableCreatedAt, CircleOrderableName, CircleOrderableNameid:
		return true
	}
	return false
}

func (e CircleOrderable) String() string {
	return string(e)
}

func (e *CircleOrderable) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = CircleOrderable(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid CircleOrderable", str)
	}
	return nil
}

func (e CircleOrderable) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type DgraphIndex string

const (
	DgraphIndexInt      DgraphIndex = "int"
	DgraphIndexFloat    DgraphIndex = "float"
	DgraphIndexBool     DgraphIndex = "bool"
	DgraphIndexHash     DgraphIndex = "hash"
	DgraphIndexExact    DgraphIndex = "exact"
	DgraphIndexTerm     DgraphIndex = "term"
	DgraphIndexFulltext DgraphIndex = "fulltext"
	DgraphIndexTrigram  DgraphIndex = "trigram"
	DgraphIndexRegexp   DgraphIndex = "regexp"
	DgraphIndexYear     DgraphIndex = "year"
	DgraphIndexMonth    DgraphIndex = "month"
	DgraphIndexDay      DgraphIndex = "day"
	DgraphIndexHour     DgraphIndex = "hour"
)

var AllDgraphIndex = []DgraphIndex{
	DgraphIndexInt,
	DgraphIndexFloat,
	DgraphIndexBool,
	DgraphIndexHash,
	DgraphIndexExact,
	DgraphIndexTerm,
	DgraphIndexFulltext,
	DgraphIndexTrigram,
	DgraphIndexRegexp,
	DgraphIndexYear,
	DgraphIndexMonth,
	DgraphIndexDay,
	DgraphIndexHour,
}

func (e DgraphIndex) IsValid() bool {
	switch e {
	case DgraphIndexInt, DgraphIndexFloat, DgraphIndexBool, DgraphIndexHash, DgraphIndexExact, DgraphIndexTerm, DgraphIndexFulltext, DgraphIndexTrigram, DgraphIndexRegexp, DgraphIndexYear, DgraphIndexMonth, DgraphIndexDay, DgraphIndexHour:
		return true
	}
	return false
}

func (e DgraphIndex) String() string {
	return string(e)
}

func (e *DgraphIndex) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = DgraphIndex(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid DgraphIndex", str)
	}
	return nil
}

func (e DgraphIndex) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type MandateOrderable string

const (
	MandateOrderableCreatedAt        MandateOrderable = "createdAt"
	MandateOrderableMessage          MandateOrderable = "message"
	MandateOrderablePurpose          MandateOrderable = "purpose"
	MandateOrderableResponsabilities MandateOrderable = "responsabilities"
	MandateOrderableDomains          MandateOrderable = "domains"
)

var AllMandateOrderable = []MandateOrderable{
	MandateOrderableCreatedAt,
	MandateOrderableMessage,
	MandateOrderablePurpose,
	MandateOrderableResponsabilities,
	MandateOrderableDomains,
}

func (e MandateOrderable) IsValid() bool {
	switch e {
	case MandateOrderableCreatedAt, MandateOrderableMessage, MandateOrderablePurpose, MandateOrderableResponsabilities, MandateOrderableDomains:
		return true
	}
	return false
}

func (e MandateOrderable) String() string {
	return string(e)
}

func (e *MandateOrderable) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = MandateOrderable(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid MandateOrderable", str)
	}
	return nil
}

func (e MandateOrderable) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type NodeOrderable string

const (
	NodeOrderableCreatedAt NodeOrderable = "createdAt"
	NodeOrderableName      NodeOrderable = "name"
	NodeOrderableNameid    NodeOrderable = "nameid"
)

var AllNodeOrderable = []NodeOrderable{
	NodeOrderableCreatedAt,
	NodeOrderableName,
	NodeOrderableNameid,
}

func (e NodeOrderable) IsValid() bool {
	switch e {
	case NodeOrderableCreatedAt, NodeOrderableName, NodeOrderableNameid:
		return true
	}
	return false
}

func (e NodeOrderable) String() string {
	return string(e)
}

func (e *NodeOrderable) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = NodeOrderable(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid NodeOrderable", str)
	}
	return nil
}

func (e NodeOrderable) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type NodeType string

const (
	NodeTypeCircle NodeType = "Circle"
	NodeTypeRole   NodeType = "Role"
)

var AllNodeType = []NodeType{
	NodeTypeCircle,
	NodeTypeRole,
}

func (e NodeType) IsValid() bool {
	switch e {
	case NodeTypeCircle, NodeTypeRole:
		return true
	}
	return false
}

func (e NodeType) String() string {
	return string(e)
}

func (e *NodeType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = NodeType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid NodeType", str)
	}
	return nil
}

func (e NodeType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type PostOrderable string

const (
	PostOrderableCreatedAt PostOrderable = "createdAt"
	PostOrderableMessage   PostOrderable = "message"
)

var AllPostOrderable = []PostOrderable{
	PostOrderableCreatedAt,
	PostOrderableMessage,
}

func (e PostOrderable) IsValid() bool {
	switch e {
	case PostOrderableCreatedAt, PostOrderableMessage:
		return true
	}
	return false
}

func (e PostOrderable) String() string {
	return string(e)
}

func (e *PostOrderable) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = PostOrderable(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid PostOrderable", str)
	}
	return nil
}

func (e PostOrderable) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type RoleOrderable string

const (
	RoleOrderableCreatedAt RoleOrderable = "createdAt"
	RoleOrderableName      RoleOrderable = "name"
	RoleOrderableNameid    RoleOrderable = "nameid"
	RoleOrderableSkills    RoleOrderable = "skills"
)

var AllRoleOrderable = []RoleOrderable{
	RoleOrderableCreatedAt,
	RoleOrderableName,
	RoleOrderableNameid,
	RoleOrderableSkills,
}

func (e RoleOrderable) IsValid() bool {
	switch e {
	case RoleOrderableCreatedAt, RoleOrderableName, RoleOrderableNameid, RoleOrderableSkills:
		return true
	}
	return false
}

func (e RoleOrderable) String() string {
	return string(e)
}

func (e *RoleOrderable) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = RoleOrderable(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid RoleOrderable", str)
	}
	return nil
}

func (e RoleOrderable) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type TensionOrderable string

const (
	TensionOrderableCreatedAt TensionOrderable = "createdAt"
	TensionOrderableMessage   TensionOrderable = "message"
	TensionOrderableNth       TensionOrderable = "nth"
	TensionOrderableTitle     TensionOrderable = "title"
	TensionOrderableSeverity  TensionOrderable = "severity"
)

var AllTensionOrderable = []TensionOrderable{
	TensionOrderableCreatedAt,
	TensionOrderableMessage,
	TensionOrderableNth,
	TensionOrderableTitle,
	TensionOrderableSeverity,
}

func (e TensionOrderable) IsValid() bool {
	switch e {
	case TensionOrderableCreatedAt, TensionOrderableMessage, TensionOrderableNth, TensionOrderableTitle, TensionOrderableSeverity:
		return true
	}
	return false
}

func (e TensionOrderable) String() string {
	return string(e)
}

func (e *TensionOrderable) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = TensionOrderable(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid TensionOrderable", str)
	}
	return nil
}

func (e TensionOrderable) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type TensionType string

const (
	TensionTypeGovernance  TensionType = "Governance"
	TensionTypeOperational TensionType = "Operational"
	TensionTypePersonal    TensionType = "Personal"
	TensionTypeHelp        TensionType = "Help"
	TensionTypeAlert       TensionType = "Alert"
)

var AllTensionType = []TensionType{
	TensionTypeGovernance,
	TensionTypeOperational,
	TensionTypePersonal,
	TensionTypeHelp,
	TensionTypeAlert,
}

func (e TensionType) IsValid() bool {
	switch e {
	case TensionTypeGovernance, TensionTypeOperational, TensionTypePersonal, TensionTypeHelp, TensionTypeAlert:
		return true
	}
	return false
}

func (e TensionType) String() string {
	return string(e)
}

func (e *TensionType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = TensionType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid TensionType", str)
	}
	return nil
}

func (e TensionType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type UserOrderable string

const (
	UserOrderableCreatedAt UserOrderable = "createdAt"
	UserOrderableUsername  UserOrderable = "username"
	UserOrderableFullname  UserOrderable = "fullname"
	UserOrderablePassword  UserOrderable = "password"
	UserOrderableBio       UserOrderable = "bio"
)

var AllUserOrderable = []UserOrderable{
	UserOrderableCreatedAt,
	UserOrderableUsername,
	UserOrderableFullname,
	UserOrderablePassword,
	UserOrderableBio,
}

func (e UserOrderable) IsValid() bool {
	switch e {
	case UserOrderableCreatedAt, UserOrderableUsername, UserOrderableFullname, UserOrderablePassword, UserOrderableBio:
		return true
	}
	return false
}

func (e UserOrderable) String() string {
	return string(e)
}

func (e *UserOrderable) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = UserOrderable(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid UserOrderable", str)
	}
	return nil
}

func (e UserOrderable) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}
