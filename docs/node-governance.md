# Node governance

A governance tension governs at most one Node through the backend-owned `Tension.governed_node` relation. Its `receiver` remains the parent circle or addressee; it is never a substitute for the governed Node.

## State and operations

The latest Node blob is the proposed document. Governance state is derived rather than persisted separately:

- **Draft:** the latest blob has a Node fragment and `governed_node` is null. The fragment's `type_` supplies the draft kind.
- **Published:** `governed_node` is set and its `isArchived` is false. The Node's `type_` supplies the kind.
- **Archived:** `governed_node.isArchived` is true (or `isRootArchived` for a root Node, see below).

`BlobPushed` creates and links a draft Node on first publication, then updates only that linked Node. Linking writes `Node.source` and `Tension.governed_node` in a single upsert. A publication interrupted between Node creation and linking leaves an orphan Node; it is repaired by the same `Node.source -> Blob.tension` upgrade script used for the migration (`../db/script/upgrade/to-v0.9.0.sh`), not at runtime. Governance requires a blob with a node fragment.

`BlobArchived` and `BlobUnarchived` drive lifecycle transitions, writing `Node.isArchived` in one upsert. First-link cleanup remains non-blocking, but runs only after archive persistence succeeds.

**Recursive archive.** Archiving a non-root node archives its whole subtree in one upsert (`db.ArchiveNodesRecursive`, `@recurse` over non-archived `Node.children`), and unlinks every first-link found. Unarchive is single-node and refuses a node whose parent is archived.

**Subtree authority.** Because the archive recurses, authority on the target (EMAP `TargetCoordoHook`, i.e. the tension receiver) is not enough: the user must also have authority on *every* non-archived descendant circle. `auth.HasSubtreeCoordoAuth` (`web/auth/gbac.go`) walks `db.GetSubNodeVisibilities` and runs `HasCoordoAuth` per circle, returning the first blocking nameid. `TryChangeArchiveNode` calls it before any write, so a denial leaves the tree untouched. This is deliberately *not* an EMAP hook: `em.Auth` flags are OR-combined, so a conjunctive constraint would never be enforced once `TargetCoordoHook` grants. A root archive is exempt (it is a flag, no recursion). Unarchive is unaffected.

**Bulk tension close.** When `BlobArchived` carries `event.new == "true"`, `ChangeArchiveBlob` closes every Open tension received by the archived subtree through the normal event pipeline (`closeSubtreeTensions` -> `ProcessEvent` with `doCheck=false`, then `PublishTensionEvent`): status propagation, history and email notifications, one per tension. Cost is linear in the number of open tensions, uncapped. Close errors are logged and the first one is returned, without rolling back the archive.

**Root archive.** Archiving an organisation (root Node) is a lightweight flag: `TryChangeArchiveNode` branches on `codec.IsRoot(nameid)` and writes `Node.isRootArchived` only — no children check, no descendant recursion, no first-link unlink, `Node.isArchived` stays false so children and tensions keep working. The lifecycle gate reads the same flag for roots (`isNodeArchived` in `graph/tension_governance.go`), so double archive/unarchive still errors. Clients (breadcrumb tag, profile active/archived tabs) read `Node.isRootArchived`.
 Authority, visibility and membership events also target the governed relation. `Node.isArchived` is the lifecycle source of truth. `Moved` is the exception: it resolves a governance subject (and re-parents the Node) only when the tension has a `governed_node`; an ordinary tension just changes receiver. Both stay inside the same organisation.

`Node.updatedAt` is bumped only by events that write the Node (publish, archive/unarchive, authority, visibility, membership, move); commenting or labelling a governance tension leaves it untouched. State and shape validation is concentrated in one pure, DB-free gate: `resolveGovernanceSubject` (`graph/tension_governance.go`) checks fragment shape, name and nameid validity, kind agreement between fragment and Node, and legal lifecycle transitions, then returns a `governanceSubject` (blob, governed Node, target nameid, create flag). The `Try*` operations in `graph/node_op.go` trust that subject: they share a uniform `(uctx, tension, subject, extras...) error` signature and own the full Node + blob + link DB transition (node write, `governed_node` linking, `Blob.pushedFlag` on publish). EMAP actions (`graph/tension_op.go`) are `func(uctx, tension, event) error` and only orchestrate resolver plus operation. DQL mutations stay plain writes, relying on the GraphQL schema for edge consistency instead of re-checking it.

**Frontend probe.** `POST /q/nodes/subauth` (`handlers.SubNodeAuth`) returns a bare bool telling whether the user has subtree authority on a nameid, so the archive modal can adapt its labels and disable the action. It is advisory only — `TryChangeArchiveNode` remains the gate and its error (naming the blocking circle) stays user-visible.

Root and spreadsheet-imported governance tensions use the same linking operation for their already-created Nodes. REST tension lists (`/q/tensions/{int,ext,all}`) expose the compact governed Node and latest fragment kind so clients can derive the same state. The `light` list keeps its smaller contract.

