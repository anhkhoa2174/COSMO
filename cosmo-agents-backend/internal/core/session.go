package core

import (
	"context"
	"database/sql"

	"gorm.io/gorm"
)

// Session aims at facilitating business transactions while abstracting the underlying mechanism,
// be it a database transaction or another transaction mechanism. This allows services to execute
// multiple business use-cases and easily rollback changes in case of error, without creating a
// dependency to the database layer.
//
// Sessions should be constituted of a root session created with a "New"-type constructor and allow
// the creation of child sessions with `Begin()` and `Transaction()`. Nested transactions should be supported
// as well.
type Session interface {
	// Begin returns a new session with the given context and a started transaction.
	// Using the returned session should have no side-effect on the parent session.
	// The underlying transaction mechanism is injected as a value into the new session's context.
	Begin(ctx context.Context) (Session, error)

	// Rollback the changes in the transaction. This action is final.
	Rollback() error

	// Commit the changes in the transaction. This action is final.
	Commit() error

	// Context returns the session's context. If it's the root session, `context.Background()` is returned.
	// If it's a child session started with `Begin()`, then the context will contain the associated
	// transaction mechanism as a value.
	Context() context.Context
}

// Gorm session implementation.
type Gorm struct {
	db        *gorm.DB
	TxOptions *sql.TxOptions
	ctx       context.Context
}

// NewGormSession create a new root session for Gorm.
// The transaction options are optional.
func NewGormSession(db *gorm.DB, opt *sql.TxOptions) Gorm {
	return Gorm{
		db:        db,
		TxOptions: opt,
		ctx:       context.Background(),
	}
}

// Begin returns a new session with the given context and a started DB transaction.
// The returned session has manual controls. Make sure a call to `Rollback()` or `Commit()`
// is executed before the session is expired (eligible for garbage collection).
// The Gorm DB associated with this session is injected as a value into the new session's context.
// If a Gorm DB is found in the given context, it will be used instead of this Session's DB, allowing for
// nested transactions.
func (s Gorm) Begin(ctx context.Context) (Session, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	tx := DB(ctx, s.db).WithContext(ctx).Begin(s.TxOptions)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return Gorm{
		ctx:       context.WithValue(ctx, dbKey{}, tx),
		TxOptions: s.TxOptions,
		db:        tx,
	}, nil
}

// Rollback the changes in the transaction. This action is final.
func (s Gorm) Rollback() error {
	return s.db.Rollback().Error
}

// Commit the changes in the transaction. This action is final.
func (s Gorm) Commit() error {
	return s.db.Commit().Error
}

// Context returns the session's context. If it's the root session, `context.Background()`
// is returned. If it's a child session started with `Begin()`, then the context will contain
// the associated Gorm DB and can be used in combination with `session.DB()`.
func (s Gorm) Context() context.Context {
	return s.ctx
}

// dbKey the key used to store the database in the context.
type dbKey struct{}

// DB returns the Gorm instance stored in the given context. Returns the given fallback
// if no Gorm DB could be found in the context.
func DB(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	db, ok := ctx.Value(dbKey{}).(*gorm.DB)
	if !ok {
		return fallback
	}
	return db
}
