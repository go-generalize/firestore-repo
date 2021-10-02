// Code generated by firestore-repo. DO NOT EDIT.
// generated version: 1.8.1
package examples

import (
	"context"
	"crypto/md5"
	"fmt"
	"reflect"
	"strings"

	"cloud.google.com/go/firestore"
	"golang.org/x/xerrors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Unique - Collections for unique constraints
type Unique struct {
	ID         string `firestore:"-"`
	Collection string
	Data       string
	Value      string
}

// UniqueRepositoryMiddleware - middleware
type UniqueRepositoryMiddleware interface {
	WrapError(ctx context.Context, err error, uniques []*Unique) error
}

// RollbackFunc - rollback function
type RollbackFunc func(context.Context) error

type uniqueRepository struct {
	collectionName   string
	targetCollection string
	firestoreClient  *firestore.Client
	middleware       []UniqueRepositoryMiddleware
}

func newUniqueRepository(firestoreClient *firestore.Client, collection string) *uniqueRepository {
	return &uniqueRepository{
		collectionName:   "Unique",
		targetCollection: collection,
		firestoreClient:  firestoreClient,
	}
}

func (repo *uniqueRepository) setMiddleware(ctx context.Context) {
	if len(repo.middleware) > 0 {
		return
	}
	if m, ok := ctx.Value(UniqueMiddlewareKey{}).(UniqueRepositoryMiddleware); ok {
		repo.middleware = append(repo.middleware, m)
	}
}

func (repo *uniqueRepository) wrapError(ctx context.Context, err error, uniques []*Unique) error {
	for _, m := range repo.middleware {
		if err = m.WrapError(ctx, err, uniques); err != nil {
			return xerrors.Errorf("wrap error middleware: %w", err)
		}
	}
	return err
}

func (repo *uniqueRepository) getUniqueItems(value interface{}) ([]*Unique, bool) {
	rv := reflect.Indirect(reflect.ValueOf(value))
	rt := rv.Type()
	uniques := make([]*Unique, 0)
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		fieldName := f.Name

		field := rv.FieldByName(fieldName)
		if !field.IsValid() {
			continue
		}

		if f.Anonymous {
			if items, ok := repo.getUniqueItems(field.Interface()); ok {
				uniques = append(uniques, items...)
			}
			continue
		}

		if field.Kind() != reflect.String {
			continue
		}

		fieldValue := field.String()
		if len(fieldValue) == 0 {
			continue
		}

		if _, hasUnique := f.Tag.Lookup("unique"); !hasUnique {
			continue
		}

		keySet := []string{
			repo.collectionName,
			fieldName,
			fieldValue,
		}
		key := strings.Join(keySet, "#")

		u := &Unique{
			ID:         fmt.Sprintf("%x", md5.Sum([]byte(key))),
			Collection: repo.collectionName,
			Data:       fieldName,
			Value:      fieldValue,
		}

		uniques = append(uniques, u)
	}

	return uniques, len(uniques) > 0
}

// CheckUnique - unique constraint check(Insert/Update)
func (repo *uniqueRepository) CheckUnique(ctx context.Context, old, subject interface{}) (RollbackFunc, error) {
	if subject == nil {
		return nil, xerrors.New("nil is not allowed")
	} else if ctx.Value(transactionInProgressKey{}) != nil {
		ctx = context.Background()
	}

	rbInsert := make([]Unique, 0)
	rbDelete := make([]Unique, 0)

	if err := repo.firestoreClient.RunTransaction(ctx, func(cx context.Context, tx *firestore.Transaction) (err error) {
		sustain := make(map[Unique]struct{}, 0)
		deleteTarget := make(map[Unique]struct{}, 0)

		switch {
		case old != nil:
			if rt1, rt2 := reflect.TypeOf(old), reflect.TypeOf(subject); rt1 != rt2 {
				return xerrors.Errorf("different type: %v != %v", rt1, rt2)
			}

			items, ok := repo.getUniqueItems(old)
			if !ok {
				break
			}

			for _, unique := range items {
				u, err := repo.get(tx, unique.ID)
				if err != nil {
					if xerrors.Is(err, ErrNotFound) {
						continue
					}
					return xerrors.Errorf("failed to get: %w", err)
				}

				if _, ok := sustain[*u]; !ok {
					sustain[*u] = struct{}{}
				}

				if _, ok := deleteTarget[*u]; !ok {
					deleteTarget[*u] = struct{}{}
				}
			}
		}

		items, ok := repo.getUniqueItems(subject)
		if !ok {
			// NOTE(54m): If the struct does not have a `unique` tag
			return nil
		}

		surveyTarget := make([]*Unique, 0)
		for _, u := range items {
			if _, ok := deleteTarget[*u]; ok {
				delete(deleteTarget, *u)
				continue
			}

			// NOTE(54m): Extract those that violate unique constraints, excluding existing ones
			if _, ok := sustain[*u]; !ok {
				surveyTarget = append(surveyTarget, u)
			}
		}

		// NOTE(54m): Check unique items
		{
			uniques := make([]*Unique, 0)
			for _, unique := range surveyTarget {
				u, err := repo.get(tx, unique.ID)
				if err != nil {
					if xerrors.Is(err, ErrNotFound) {
						continue
					}
					return xerrors.Errorf("failed to get(Tx): %w", err)
				}

				uniques = append(uniques, u)
			}

			if len(uniques) > 0 {
				return repo.wrapError(cx, ErrUniqueConstraint, uniques)
			}
		}

		// NOTE(54m): Delete unique items
		for u := range deleteTarget {
			delete(sustain, u)
			rbInsert = append(rbInsert, u)
			if err := repo.deleteByID(tx, u.ID); err != nil {
				return xerrors.Errorf("failed to delete(Tx): %w", err)
			}
		}

		// NOTE(54m): Insert unique item
		for _, u := range surveyTarget {
			rbDelete = append(rbDelete, *u)
			if err := repo.insert(tx, u); err != nil {
				return xerrors.Errorf("failed to insert(Tx): %w", err)
			}
		}

		return nil
	}); err != nil {
		return nil, xerrors.Errorf("could not run transaction: %w", err)
	}

	return repo.makeRollbackFunc(rbInsert, rbDelete), nil
}

// DeleteUnique - delete the document for unique constraints
func (repo *uniqueRepository) DeleteUnique(ctx context.Context, subject interface{}) (RollbackFunc, error) {
	if subject == nil {
		return nil, xerrors.New("nil is not allowed")
	} else if ctx.Value(transactionInProgressKey{}) != nil {
		ctx = context.Background()
	}

	rbInsert := make([]Unique, 0)

	if err := repo.firestoreClient.RunTransaction(ctx, func(cx context.Context, tx *firestore.Transaction) error {
		deleteTarget := make(map[Unique]struct{}, 0)

		items, ok := repo.getUniqueItems(subject)
		if !ok {
			return nil
		}

		for _, unique := range items {
			u, err := repo.get(tx, unique.ID)
			if err != nil {
				if xerrors.Is(err, ErrNotFound) {
					continue
				}
				return xerrors.Errorf("failed to get: %w", err)
			}

			if _, ok := deleteTarget[*u]; !ok {
				deleteTarget[*u] = struct{}{}
			}
		}

		// NOTE(54m): Delete unique items
		for u := range deleteTarget {
			rbInsert = append(rbInsert, u)
			if err := repo.deleteByID(tx, u.ID); err != nil {
				return xerrors.Errorf("failed to delete(Tx): %w", err)
			}
		}

		return nil
	}); err != nil {
		return nil, xerrors.Errorf("could not run transaction: %w", err)
	}

	return repo.makeRollbackFunc(rbInsert, []Unique{}), nil
}

func (repo *uniqueRepository) makeRollbackFunc(rbInsert, rbDelete []Unique) RollbackFunc {
	return func(ctx context.Context) error {
		if ctx.Value(transactionInProgressKey{}) != nil {
			ctx = context.Background()
		}
		if err := repo.firestoreClient.RunTransaction(ctx, func(cx context.Context, tx *firestore.Transaction) error {
			for _, unique := range rbInsert {
				// escape: scopelint
				if err := repo.insert(tx, &Unique{
					ID:         unique.ID,
					Collection: unique.Collection,
					Data:       unique.Data,
					Value:      unique.Value,
				}); err != nil {
					return xerrors.Errorf("failed to insert(Tx): %w", err)
				}
			}
			for _, unique := range rbDelete {
				if err := repo.deleteByID(tx, unique.ID); err != nil {
					return xerrors.Errorf("failed to delete(Tx): %w", err)
				}
			}
			return nil
		}); err != nil {
			return xerrors.Errorf("could not run rollback(unique): %w", err)
		}
		return nil
	}
}

func (repo *uniqueRepository) getCollection() *firestore.CollectionRef {
	return repo.firestoreClient.Collection(repo.collectionName)
}

func (repo *uniqueRepository) getDocRef(id string) *firestore.DocumentRef {
	return repo.getCollection().Doc(id)
}

func (repo *uniqueRepository) get(tx *firestore.Transaction, id string) (*Unique, error) {
	doc := repo.getDocRef(id)

	snapShot, err := tx.Get(doc)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, xerrors.Errorf("error in Get method(Tx): %w", err)
	}

	subject := new(Unique)
	if err := snapShot.DataTo(&subject); err != nil {
		return nil, xerrors.Errorf("error in DataTo method: %w", err)
	}
	subject.ID = snapShot.Ref.ID

	return subject, nil
}

func (repo *uniqueRepository) insert(tx *firestore.Transaction, subject *Unique) error {
	dr := repo.getDocRef(subject.ID)

	if err := tx.Set(dr, subject); err != nil {
		return xerrors.Errorf("error in Set method(Tx): %w", err)
	}
	subject.ID = dr.ID

	return nil
}

func (repo *uniqueRepository) deleteByID(tx *firestore.Transaction, id string) error {
	dr := repo.getDocRef(id)

	if err := tx.Delete(dr, firestore.Exists); err != nil {
		return xerrors.Errorf("error in Delete method(Tx): %w", err)
	}

	return nil
}
