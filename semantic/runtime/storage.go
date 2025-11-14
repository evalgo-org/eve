package runtime

// storage.go defines storage interfaces for RuntimeAction and Event persistence.
// Implementations:
//   - RuntimeRepository (repository.go): CouchDB storage for RuntimeActions
//   - EventStore (event_store.go): PostgreSQL storage for Events
//
// These follow EVE's repository pattern established in eve.evalgo.org/db/repository
