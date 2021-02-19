// Code generated by firestore-repo. DO NOT EDIT.
// generated version: 0.9.1
package examples

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	model "github.com/go-generalize/firestore-repo/examples"
	"golang.org/x/xerrors"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//go:generate mockgen -source lock_gen.go -destination mock/mock_lock_gen/mock_lock_gen.go

// LockRepository - Repository of Lock
type LockRepository interface {
	// Single
	Get(ctx context.Context, id string, opts ...GetOption) (*model.Lock, error)
	GetWithDoc(ctx context.Context, doc *firestore.DocumentRef, opts ...GetOption) (*model.Lock, error)
	Insert(ctx context.Context, subject *model.Lock) (_ string, err error)
	Update(ctx context.Context, subject *model.Lock) (err error)
	StrictUpdate(ctx context.Context, id string, param *LockUpdateParam, opts ...firestore.Precondition) error
	Delete(ctx context.Context, subject *model.Lock, opts ...DeleteOption) (err error)
	DeleteByID(ctx context.Context, id string, opts ...DeleteOption) (err error)
	// Multiple
	GetMulti(ctx context.Context, ids []string, opts ...GetOption) ([]*model.Lock, error)
	InsertMulti(ctx context.Context, subjects []*model.Lock) (_ []string, er error)
	UpdateMulti(ctx context.Context, subjects []*model.Lock) (er error)
	DeleteMulti(ctx context.Context, subjects []*model.Lock, opts ...DeleteOption) (er error)
	DeleteMultiByIDs(ctx context.Context, ids []string, opts ...DeleteOption) (er error)
	// Single(Transaction)
	GetWithTx(tx *firestore.Transaction, id string, opts ...GetOption) (*model.Lock, error)
	GetWithDocWithTx(tx *firestore.Transaction, doc *firestore.DocumentRef, opts ...GetOption) (*model.Lock, error)
	InsertWithTx(ctx context.Context, tx *firestore.Transaction, subject *model.Lock) (_ string, err error)
	UpdateWithTx(ctx context.Context, tx *firestore.Transaction, subject *model.Lock) (err error)
	StrictUpdateWithTx(tx *firestore.Transaction, id string, param *LockUpdateParam, opts ...firestore.Precondition) error
	DeleteWithTx(ctx context.Context, tx *firestore.Transaction, subject *model.Lock, opts ...DeleteOption) (err error)
	DeleteByIDWithTx(ctx context.Context, tx *firestore.Transaction, id string, opts ...DeleteOption) (err error)
	// Multiple(Transaction)
	GetMultiWithTx(tx *firestore.Transaction, ids []string, opts ...GetOption) ([]*model.Lock, error)
	InsertMultiWithTx(ctx context.Context, tx *firestore.Transaction, subjects []*model.Lock) (_ []string, er error)
	UpdateMultiWithTx(ctx context.Context, tx *firestore.Transaction, subjects []*model.Lock) (er error)
	DeleteMultiWithTx(ctx context.Context, tx *firestore.Transaction, subjects []*model.Lock, opts ...DeleteOption) (er error)
	DeleteMultiByIDsWithTx(ctx context.Context, tx *firestore.Transaction, ids []string, opts ...DeleteOption) (er error)
	// List
	List(ctx context.Context, req *LockListReq, q *firestore.Query) ([]*model.Lock, error)
	ListWithTx(tx *firestore.Transaction, req *LockListReq, q *firestore.Query) ([]*model.Lock, error)
	// misc
	GetCollection() *firestore.CollectionRef
	GetCollectionName() string
	GetDocRef(id string) *firestore.DocumentRef
}

// LockRepositoryMiddleware - middleware of LockRepository
type LockRepositoryMiddleware interface {
	BeforeInsert(ctx context.Context, subject *model.Lock) (bool, error)
	BeforeUpdate(ctx context.Context, old, subject *model.Lock) (bool, error)
	BeforeDelete(ctx context.Context, subject *model.Lock, opts ...DeleteOption) (bool, error)
	BeforeDeleteByID(ctx context.Context, ids []string, opts ...DeleteOption) (bool, error)
}

type lockRepository struct {
	collectionName   string
	firestoreClient  *firestore.Client
	middleware       []LockRepositoryMiddleware
	uniqueRepository *uniqueRepository
}

// NewLockRepository - constructor
func NewLockRepository(firestoreClient *firestore.Client, middleware ...LockRepositoryMiddleware) LockRepository {
	return &lockRepository{
		collectionName:   "Lock",
		firestoreClient:  firestoreClient,
		middleware:       middleware,
		uniqueRepository: newUniqueRepository(firestoreClient, "Lock"),
	}
}

func (repo *lockRepository) setMeta(subject *model.Lock, isInsert bool) {
	now := time.Now()

	if isInsert {
		subject.CreatedAt = time.Now()
	}
	subject.UpdatedAt = now
	subject.Version++
}

func (repo *lockRepository) beforeInsert(ctx context.Context, subject *model.Lock) (RollbackFunc, error) {
	if subject.Version != 0 {
		return nil, xerrors.Errorf("insert data must be Version == 0: %+v", subject)
	}
	if subject.DeletedAt != nil {
		return nil, xerrors.Errorf("insert data must be DeletedAt == nil: %+v", subject)
	}
	repo.setMeta(subject, true)
	repo.uniqueRepository.setMiddleware(ctx)
	rb, err := repo.uniqueRepository.CheckUnique(ctx, nil, subject)
	if err != nil {
		return nil, xerrors.Errorf("unique.middleware error: %w", err)
	}

	for _, m := range repo.middleware {
		c, err := m.BeforeInsert(ctx, subject)
		if err != nil {
			return nil, xerrors.Errorf("beforeInsert.middleware error(uniqueRB=%t): %w", rb(ctx) == nil, err)
		}
		if !c {
			continue
		}
	}

	return rb, nil
}

func (repo *lockRepository) beforeUpdate(ctx context.Context, old, subject *model.Lock) (RollbackFunc, error) {
	if old.Version > subject.Version {
		return nil, xerrors.Errorf(
			"The data in the database is newer: (db version: %d, target version: %d) %+v",
			old.Version, subject.Version, subject,
		)
	}
	if subject.DeletedAt != nil {
		return nil, xerrors.Errorf("update data must be DeletedAt == nil: %+v", subject)
	}
	repo.setMeta(subject, false)
	repo.uniqueRepository.setMiddleware(ctx)
	rb, err := repo.uniqueRepository.CheckUnique(ctx, old, subject)
	if err != nil {
		return nil, xerrors.Errorf("unique.middleware error: %w", err)
	}

	for _, m := range repo.middleware {
		c, err := m.BeforeUpdate(ctx, old, subject)
		if err != nil {
			return nil, xerrors.Errorf("beforeUpdate.middleware error: %w", err)
		}
		if !c {
			continue
		}
	}

	return rb, nil
}

func (repo *lockRepository) beforeDelete(ctx context.Context, subject *model.Lock, opts ...DeleteOption) (RollbackFunc, error) {
	repo.setMeta(subject, false)
	repo.uniqueRepository.setMiddleware(ctx)
	rb, err := repo.uniqueRepository.DeleteUnique(ctx, subject)
	if err != nil {
		return nil, xerrors.Errorf("unique.middleware error: %w", err)
	}

	for _, m := range repo.middleware {
		c, err := m.BeforeDelete(ctx, subject, opts...)
		if err != nil {
			return nil, xerrors.Errorf("beforeDelete.middleware error: %w", err)
		}
		if !c {
			continue
		}
	}

	return rb, nil
}

// GetCollection - *firestore.CollectionRef getter
func (repo *lockRepository) GetCollection() *firestore.CollectionRef {
	return repo.firestoreClient.Collection(repo.collectionName)
}

// GetCollectionName - CollectionName getter
func (repo *lockRepository) GetCollectionName() string {
	return repo.collectionName
}

// GetDocRef  - *firestore.DocumentRef getter
func (repo *lockRepository) GetDocRef(id string) *firestore.DocumentRef {
	return repo.GetCollection().Doc(id)
}

// LockListReq - params for search
type LockListReq struct {
	Text      *QueryChainer
	Flag      *QueryChainer
	CreatedAt *QueryChainer
	CreatedBy *QueryChainer
	DeletedAt *QueryChainer
	DeletedBy *QueryChainer
	UpdatedAt *QueryChainer
	UpdatedBy *QueryChainer
	Version   *QueryChainer

	IncludeSoftDeleted bool
}

// LockUpdateParam - params for strict updates
type LockUpdateParam struct {
	Flag      interface{}
	CreatedAt interface{}
	CreatedBy interface{}
	DeletedAt interface{}
	DeletedBy interface{}
	UpdatedAt interface{}
	UpdatedBy interface{}
	Version   interface{}
}

// List - search documents
// The third argument is firestore.Query, basically you can pass nil
func (repo *lockRepository) List(ctx context.Context, req *LockListReq, q *firestore.Query) ([]*model.Lock, error) {
	return repo.list(ctx, req, q)
}

// Get - get `Lock` by `Lock.ID`
func (repo *lockRepository) Get(ctx context.Context, id string, opts ...GetOption) (*model.Lock, error) {
	doc := repo.GetDocRef(id)
	return repo.get(ctx, doc, opts...)
}

// GetWithDoc - get `Lock` by *firestore.DocumentRef
func (repo *lockRepository) GetWithDoc(ctx context.Context, doc *firestore.DocumentRef, opts ...GetOption) (*model.Lock, error) {
	return repo.get(ctx, doc, opts...)
}

// Insert - insert of `Lock`
func (repo *lockRepository) Insert(ctx context.Context, subject *model.Lock) (_ string, err error) {
	rb, err := repo.beforeInsert(ctx, subject)
	if err != nil {
		return "", xerrors.Errorf("before insert error: %w", err)
	}
	defer func() {
		if err != nil {
			if er := rb(ctx); er != nil {
				err = xerrors.Errorf("unique check error %+v, original error: %w", er, err)
			}
		}
	}()

	return repo.insert(ctx, subject)
}

// Update - update of `Lock`
func (repo *lockRepository) Update(ctx context.Context, subject *model.Lock) (err error) {
	doc := repo.GetDocRef(subject.ID)

	old, err := repo.get(ctx, doc)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return ErrNotFound
		}
		return xerrors.Errorf("error in Get method: %w", err)
	}

	rb, err := repo.beforeUpdate(ctx, old, subject)
	if err != nil {
		return xerrors.Errorf("before update error: %w", err)
	}
	defer func() {
		if err != nil {
			if er := rb(ctx); er != nil {
				err = xerrors.Errorf("unique check error %+v, original error: %w", er, err)
			}
		}
	}()

	return repo.update(ctx, subject)
}

// StrictUpdate - strict update of `Lock`
func (repo *lockRepository) StrictUpdate(ctx context.Context, id string, param *LockUpdateParam, opts ...firestore.Precondition) error {
	return repo.strictUpdate(ctx, id, param, opts...)
}

// Delete - delete of `Lock`
func (repo *lockRepository) Delete(ctx context.Context, subject *model.Lock, opts ...DeleteOption) (err error) {
	if len(opts) > 0 && opts[0].Mode == DeleteModeSoft {
		t := time.Now()
		subject.DeletedAt = &t
		if err := repo.update(ctx, subject); err != nil {
			return xerrors.Errorf("error in update method: %w", err)
		}
		return nil
	}

	rb, err := repo.beforeDelete(ctx, subject, opts...)
	if err != nil {
		return xerrors.Errorf("before delete error: %w", err)
	}
	defer func() {
		if err != nil {
			if er := rb(ctx); er != nil {
				err = xerrors.Errorf("unique delete error %+v, original error: %w", er, err)
			}
		}
	}()

	return repo.deleteByID(ctx, subject.ID)
}

// DeleteByID - delete `Lock` by `Lock.ID`
func (repo *lockRepository) DeleteByID(ctx context.Context, id string, opts ...DeleteOption) (err error) {
	subject, err := repo.Get(ctx, id)
	if err != nil {
		return xerrors.Errorf("error in Get method: %w", err)
	}

	if len(opts) > 0 && opts[0].Mode == DeleteModeSoft {
		t := time.Now()
		subject.DeletedAt = &t
		if err := repo.update(ctx, subject); err != nil {
			return xerrors.Errorf("error in update method: %w", err)
		}
		return nil
	}

	return repo.Delete(ctx, subject, opts...)
}

// GetMulti - get `Lock` in bulk by array of `Lock.ID`
func (repo *lockRepository) GetMulti(ctx context.Context, ids []string, opts ...GetOption) ([]*model.Lock, error) {
	return repo.getMulti(ctx, ids, opts...)
}

// InsertMulti - bulk insert of `Lock`
func (repo *lockRepository) InsertMulti(ctx context.Context, subjects []*model.Lock) (_ []string, er error) {
	var rbs []RollbackFunc
	defer func() {
		if er == nil {
			return
		}
		if len(rbs) == 0 {
			return
		}
		errs := make([]error, 0)
		for _, rb := range rbs {
			if err := rb(ctx); err != nil {
				errs = append(errs, err)
			}
		}
		er = xerrors.Errorf("unique check error %+v, original error: %w", errs, er)
	}()

	ids := make([]string, 0, len(subjects))
	batches := make([]*firestore.WriteBatch, 0)
	batch := repo.firestoreClient.Batch()
	collect := repo.GetCollection()

	for i, subject := range subjects {
		var ref *firestore.DocumentRef
		if subject.ID == "" {
			ref = collect.NewDoc()
			subject.ID = ref.ID
		} else {
			ref = collect.Doc(subject.ID)
			if s, err := ref.Get(ctx); err == nil {
				return nil, xerrors.Errorf("already exists [%v]: %#v", subject.ID, s)
			}
		}

		rb, err := repo.beforeInsert(ctx, subject)
		if err != nil {
			return nil, xerrors.Errorf("before insert error(%d) [%v]: %w", i, subject.ID, err)
		}
		rbs = append(rbs, rb)

		batch.Set(ref, subject)
		ids = append(ids, ref.ID)
		i++
		if (i%500) == 0 && len(subjects) != i {
			batches = append(batches, batch)
			batch = repo.firestoreClient.Batch()
		}
	}
	batches = append(batches, batch)

	for _, b := range batches {
		if _, err := b.Commit(ctx); err != nil {
			return nil, xerrors.Errorf("error in Commit method: %w", err)
		}
	}

	return ids, nil
}

// UpdateMulti - bulk update of `Lock`
func (repo *lockRepository) UpdateMulti(ctx context.Context, subjects []*model.Lock) (er error) {
	var rbs []RollbackFunc
	defer func() {
		if er == nil {
			return
		}
		if len(rbs) == 0 {
			return
		}
		errs := make([]error, 0)
		for _, rb := range rbs {
			if err := rb(ctx); err != nil {
				errs = append(errs, err)
			}
		}
		er = xerrors.Errorf("unique check error %+v, original error: %w", errs, er)
	}()

	batches := make([]*firestore.WriteBatch, 0)
	batch := repo.firestoreClient.Batch()
	collect := repo.GetCollection()

	for i, subject := range subjects {
		ref := collect.Doc(subject.ID)
		snapShot, err := ref.Get(ctx)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return xerrors.Errorf("not found [%v]: %w", subject.ID, err)
			}
			return xerrors.Errorf("error in Get method [%v]: %w", subject.ID, err)
		}

		old := new(model.Lock)
		if err = snapShot.DataTo(&old); err != nil {
			return xerrors.Errorf("error in DataTo method: %w", err)
		}

		rb, err := repo.beforeUpdate(ctx, old, subject)
		if err != nil {
			return xerrors.Errorf("before update error(%d) [%v]: %w", i, subject.ID, err)
		}
		rbs = append(rbs, rb)

		batch.Set(ref, subject)
		i++
		if (i%500) == 0 && len(subjects) != i {
			batches = append(batches, batch)
			batch = repo.firestoreClient.Batch()
		}
	}
	batches = append(batches, batch)

	for _, b := range batches {
		if _, err := b.Commit(ctx); err != nil {
			return xerrors.Errorf("error in Commit method: %w", err)
		}
	}

	return nil
}

// DeleteMulti - bulk delete of `Lock`
func (repo *lockRepository) DeleteMulti(ctx context.Context, subjects []*model.Lock, opts ...DeleteOption) (er error) {
	var rbs []RollbackFunc
	defer func() {
		if er == nil {
			return
		}
		if len(rbs) == 0 {
			return
		}
		errs := make([]error, 0)
		for _, rb := range rbs {
			if err := rb(ctx); err != nil {
				errs = append(errs, err)
			}
		}
		er = xerrors.Errorf("unique delete error %+v, original error: %w", errs, er)
	}()

	batches := make([]*firestore.WriteBatch, 0)
	batch := repo.firestoreClient.Batch()
	collect := repo.GetCollection()

	for i, subject := range subjects {
		ref := collect.Doc(subject.ID)
		if _, err := ref.Get(ctx); err != nil {
			if status.Code(err) == codes.NotFound {
				return xerrors.Errorf("not found [%v]: %w", subject.ID, err)
			}
			return xerrors.Errorf("error in Get method [%v]: %w", subject.ID, err)
		}

		rb, err := repo.beforeDelete(ctx, subject, opts...)
		if err != nil {
			return xerrors.Errorf("before delete error(%d) [%v]: %w", i, subject.ID, err)
		}
		rbs = append(rbs, rb)

		if len(opts) > 0 && opts[0].Mode == DeleteModeSoft {
			t := time.Now()
			subject.DeletedAt = &t
			batch.Set(ref, subject)
		} else {
			batch.Delete(ref)
		}

		i++
		if (i%500) == 0 && len(subjects) != i {
			batches = append(batches, batch)
			batch = repo.firestoreClient.Batch()
		}
	}
	batches = append(batches, batch)

	for _, b := range batches {
		if _, err := b.Commit(ctx); err != nil {
			return xerrors.Errorf("error in Commit method: %w", err)
		}
	}

	return nil
}

// DeleteMultiByIDs - delete `Lock` in bulk by array of `Lock.ID`
func (repo *lockRepository) DeleteMultiByIDs(ctx context.Context, ids []string, opts ...DeleteOption) (er error) {
	subjects := make([]*model.Lock, len(ids))

	opt := GetOption{}
	if len(opts) > 0 {
		opt.IncludeSoftDeleted = opts[0].Mode == DeleteModeHard
	}
	for i, id := range ids {
		subject, err := repo.Get(ctx, id, opt)
		if err != nil {
			return xerrors.Errorf("error in Get method: %w", err)
		}
		subjects[i] = subject
	}

	return repo.DeleteMulti(ctx, subjects, opts...)
}

func (repo *lockRepository) ListWithTx(tx *firestore.Transaction, req *LockListReq, q *firestore.Query) ([]*model.Lock, error) {
	return repo.list(tx, req, q)
}

// GetWithTx - get `Lock` by `Lock.ID` in transaction
func (repo *lockRepository) GetWithTx(tx *firestore.Transaction, id string, opts ...GetOption) (*model.Lock, error) {
	doc := repo.GetDocRef(id)
	return repo.get(tx, doc, opts...)
}

// GetWithDocWithTx - get `Lock` by *firestore.DocumentRef in transaction
func (repo *lockRepository) GetWithDocWithTx(tx *firestore.Transaction, doc *firestore.DocumentRef, opts ...GetOption) (*model.Lock, error) {
	return repo.get(tx, doc, opts...)
}

// InsertWithTx - insert of `Lock` in transaction
func (repo *lockRepository) InsertWithTx(ctx context.Context, tx *firestore.Transaction, subject *model.Lock) (_ string, err error) {
	rb, err := repo.beforeInsert(context.WithValue(ctx, transactionInProgressKey{}, 1), subject)
	if err != nil {
		return "", xerrors.Errorf("before insert error: %w", err)
	}
	defer func() {
		if err != nil {
			if er := rb(ctx); er != nil {
				err = xerrors.Errorf("unique check error %+v, original error: %w", er, err)
			}
		}
	}()

	return repo.insert(tx, subject)
}

// UpdateWithTx - update of `Lock` in transaction
func (repo *lockRepository) UpdateWithTx(ctx context.Context, tx *firestore.Transaction, subject *model.Lock) (err error) {
	doc := repo.GetDocRef(subject.ID)

	old, err := repo.get(tx, doc)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return ErrNotFound
		}
		return xerrors.Errorf("error in Get method: %w", err)
	}

	rb, err := repo.beforeUpdate(context.WithValue(ctx, transactionInProgressKey{}, 1), old, subject)
	if err != nil {
		return xerrors.Errorf("before update error: %w", err)
	}
	defer func() {
		if err != nil {
			if er := rb(ctx); er != nil {
				err = xerrors.Errorf("unique check error %+v, original error: %w", er, err)
			}
		}
	}()

	return repo.update(tx, subject)
}

// StrictUpdateWithTx - strict update of `Lock` in transaction
func (repo *lockRepository) StrictUpdateWithTx(tx *firestore.Transaction, id string, param *LockUpdateParam, opts ...firestore.Precondition) error {
	return repo.strictUpdate(tx, id, param, opts...)
}

// DeleteWithTx - delete of `Lock` in transaction
func (repo *lockRepository) DeleteWithTx(ctx context.Context, tx *firestore.Transaction, subject *model.Lock, opts ...DeleteOption) (err error) {
	if len(opts) > 0 && opts[0].Mode == DeleteModeSoft {
		t := time.Now()
		subject.DeletedAt = &t
		if err := repo.update(tx, subject); err != nil {
			return xerrors.Errorf("error in update method: %w", err)
		}
		return nil
	}

	rb, err := repo.beforeDelete(ctx, subject, opts...)
	if err != nil {
		return xerrors.Errorf("before delete error: %w", err)
	}
	defer func() {
		if err != nil {
			if er := rb(ctx); er != nil {
				err = xerrors.Errorf("unique check error %+v, original error: %w", er, err)
			}
		}
	}()

	return repo.deleteByID(tx, subject.ID)
}

// DeleteByIDWithTx - delete `Lock` by `Lock.ID` in transaction
func (repo *lockRepository) DeleteByIDWithTx(ctx context.Context, tx *firestore.Transaction, id string, opts ...DeleteOption) (err error) {
	subject, err := repo.GetWithTx(tx, id)
	if err != nil {
		return xerrors.Errorf("error in GetWithTx method: %w", err)
	}

	if len(opts) > 0 && opts[0].Mode == DeleteModeSoft {
		t := time.Now()
		subject.DeletedAt = &t
		if err := repo.update(tx, subject); err != nil {
			return xerrors.Errorf("error in update method: %w", err)
		}
		return nil
	}

	rb, err := repo.beforeDelete(ctx, subject, opts...)
	if err != nil {
		return xerrors.Errorf("before delete error: %w", err)
	}
	defer func() {
		if err != nil {
			if er := rb(ctx); er != nil {
				err = xerrors.Errorf("unique delete error %+v, original error: %w", er, err)
			}
		}
	}()

	return repo.deleteByID(tx, id)
}

// GetMultiWithTx - get `Lock` in bulk by array of `Lock.ID` in transaction
func (repo *lockRepository) GetMultiWithTx(tx *firestore.Transaction, ids []string, opts ...GetOption) ([]*model.Lock, error) {
	return repo.getMulti(tx, ids, opts...)
}

// InsertMultiWithTx - bulk insert of `Lock` in transaction
func (repo *lockRepository) InsertMultiWithTx(ctx context.Context, tx *firestore.Transaction, subjects []*model.Lock) (_ []string, er error) {
	ctx = context.WithValue(ctx, transactionInProgressKey{}, 1)
	var rbs []RollbackFunc
	defer func() {
		if er == nil {
			return
		}
		if len(rbs) == 0 {
			return
		}
		errs := make([]error, 0)
		for _, rb := range rbs {
			if err := rb(ctx); err != nil {
				errs = append(errs, err)
			}
		}
		er = xerrors.Errorf("unique check error %+v, original error: %w", errs, er)
	}()

	ids := make([]string, len(subjects))

	for i := range subjects {
		rb, err := repo.beforeInsert(ctx, subjects[i])
		if err != nil {
			return nil, xerrors.Errorf("before insert error(%d) [%v]: %w", i, subjects[i].ID, err)
		}
		rbs = append(rbs, rb)

		id, err := repo.insert(tx, subjects[i])
		if err != nil {
			return nil, xerrors.Errorf("error in insert method(%d) [%v]: %w", i, subjects[i].ID, err)
		}
		ids[i] = id
	}

	return ids, nil
}

// UpdateMultiWithTx - bulk update of `Lock` in transaction
func (repo *lockRepository) UpdateMultiWithTx(ctx context.Context, tx *firestore.Transaction, subjects []*model.Lock) (er error) {
	ctx = context.WithValue(ctx, transactionInProgressKey{}, 1)
	var rbs []RollbackFunc
	defer func() {
		if er == nil {
			return
		}
		if len(rbs) == 0 {
			return
		}
		errs := make([]error, 0)
		for _, rb := range rbs {
			if err := rb(ctx); err != nil {
				errs = append(errs, err)
			}
		}
		er = xerrors.Errorf("unique check error %+v, original error: %w", errs, er)
	}()

	for i := range subjects {
		doc := repo.GetDocRef(subjects[i].ID)
		old, err := repo.get(tx, doc)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return xerrors.Errorf("not found(%d) [%v]", i, subjects[i].ID)
			}
			return xerrors.Errorf("error in get method(%d) [%v]: %w", i, subjects[i].ID, err)
		}

		rb, err := repo.beforeUpdate(ctx, old, subjects[i])
		if err != nil {
			return xerrors.Errorf("before update error(%d) [%v]: %w", i, subjects[i].ID, err)
		}
		rbs = append(rbs, rb)
	}

	for i := range subjects {
		if err := repo.update(tx, subjects[i]); err != nil {
			return xerrors.Errorf("error in update method(%d) [%v]: %w", i, subjects[i].ID, err)
		}
	}

	return nil
}

// DeleteMultiWithTx - bulk delete of `Lock` in transaction
func (repo *lockRepository) DeleteMultiWithTx(ctx context.Context, tx *firestore.Transaction, subjects []*model.Lock, opts ...DeleteOption) (er error) {
	var rbs []RollbackFunc
	defer func() {
		if er == nil {
			return
		}
		if len(rbs) == 0 {
			return
		}
		errs := make([]error, 0)
		for _, rb := range rbs {
			if err := rb(ctx); err != nil {
				errs = append(errs, err)
			}
		}
		er = xerrors.Errorf("unique delete error %+v, original error: %w", errs, er)
	}()

	t := time.Now()
	var isHardDeleteMode bool
	if len(opts) > 0 {
		isHardDeleteMode = opts[0].Mode == DeleteModeHard
	}
	opt := GetOption{
		IncludeSoftDeleted: isHardDeleteMode,
	}
	for i := range subjects {
		dr := repo.GetDocRef(subjects[i].ID)
		if _, err := repo.get(tx, dr, opt); err != nil {
			if status.Code(err) == codes.NotFound {
				return xerrors.Errorf("not found(%d) [%v]", i, subjects[i].ID)
			}
			return xerrors.Errorf("error in get method(%d) [%v]: %w", i, subjects[i].ID, err)
		}

		rb, err := repo.beforeDelete(ctx, subjects[i], opts...)
		if err != nil {
			return xerrors.Errorf("before delete error(%d) [%v]: %w", i, subjects[i].ID, err)
		}
		rbs = append(rbs, rb)

		if !isHardDeleteMode {
			subjects[i].DeletedAt = &t
			if err := repo.update(tx, subjects[i]); err != nil {
				return xerrors.Errorf("error in update method(%d) [%v]: %w", i, subjects[i].ID, err)
			}
		}
	}

	if len(opts) > 0 && opts[0].Mode == DeleteModeSoft {
		return nil
	}

	for i := range subjects {
		if err := repo.deleteByID(tx, subjects[i].ID); err != nil {
			return xerrors.Errorf("error in delete method(%d) [%v]: %w", i, subjects[i].ID, err)
		}
	}

	return nil
}

// DeleteMultiByIDWithTx - delete `Lock` in bulk by array of `Lock.ID` in transaction
func (repo *lockRepository) DeleteMultiByIDsWithTx(ctx context.Context, tx *firestore.Transaction, ids []string, opts ...DeleteOption) (er error) {
	var rbs []RollbackFunc
	defer func() {
		if er == nil {
			return
		}
		if len(rbs) == 0 {
			return
		}
		errs := make([]error, 0)
		for _, rb := range rbs {
			if err := rb(ctx); err != nil {
				errs = append(errs, err)
			}
		}
		er = xerrors.Errorf("unique delete error %+v, original error: %w", errs, er)
	}()

	t := time.Now()
	for i := range ids {
		dr := repo.GetDocRef(ids[i])
		subject, err := repo.get(tx, dr)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return xerrors.Errorf("not found(%d) [%v]", i, ids[i])
			}
			return xerrors.Errorf("error in get method(%d) [%v]: %w", i, ids[i], err)
		}

		rb, err := repo.beforeDelete(ctx, subject, opts...)
		if err != nil {
			return xerrors.Errorf("before delete error(%d) [%v]: %w", i, subject.ID, err)
		}
		rbs = append(rbs, rb)

		if len(opts) > 0 && opts[0].Mode == DeleteModeSoft {
			subject.DeletedAt = &t
			if err := repo.update(tx, subject); err != nil {
				return xerrors.Errorf("error in update method(%d) [%v]: %w", i, ids[i], err)
			}
		}
	}

	if len(opts) > 0 && opts[0].Mode == DeleteModeSoft {
		return nil
	}

	for i := range ids {
		if err := repo.deleteByID(tx, ids[i]); err != nil {
			return xerrors.Errorf("error in delete method(%d) [%v]: %w", i, ids[i], err)
		}
	}

	return nil
}

func (repo *lockRepository) get(v interface{}, doc *firestore.DocumentRef, opts ...GetOption) (*model.Lock, error) {
	var (
		snapShot *firestore.DocumentSnapshot
		err      error
	)

	switch x := v.(type) {
	case *firestore.Transaction:
		snapShot, err = x.Get(doc)
	case context.Context:
		snapShot, err = doc.Get(x)
	default:
		return nil, xerrors.Errorf("invalid type: %v", x)
	}

	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, xerrors.Errorf("error in Get method: %w", err)
	}

	subject := new(model.Lock)
	if err := snapShot.DataTo(&subject); err != nil {
		return nil, xerrors.Errorf("error in DataTo method: %w", err)
	}

	if len(opts) == 0 || !opts[0].IncludeSoftDeleted {
		if subject.DeletedAt != nil {
			return nil, ErrAlreadyDeleted
		}
	}
	subject.ID = snapShot.Ref.ID

	return subject, nil
}

func (repo *lockRepository) getMulti(v interface{}, ids []string, opts ...GetOption) ([]*model.Lock, error) {
	var (
		snapShots []*firestore.DocumentSnapshot
		err       error
		collect   = repo.GetCollection()
		drs       = make([]*firestore.DocumentRef, len(ids))
	)

	for i, id := range ids {
		ref := collect.Doc(id)
		drs[i] = ref
	}

	switch x := v.(type) {
	case *firestore.Transaction:
		snapShots, err = x.GetAll(drs)
	case context.Context:
		snapShots, err = repo.firestoreClient.GetAll(x, drs)
	default:
		return nil, xerrors.Errorf("invalid type: %v", v)
	}

	if err != nil {
		return nil, xerrors.Errorf("error in GetAll method: %w", err)
	}

	subjects := make([]*model.Lock, 0, len(ids))
	for _, snapShot := range snapShots {
		subject := new(model.Lock)
		if err := snapShot.DataTo(&subject); err != nil {
			return nil, xerrors.Errorf("error in DataTo method: %w", err)
		}

		if len(opts) == 0 || !opts[0].IncludeSoftDeleted {
			if subject.DeletedAt != nil {
				continue
			}
		}
		subject.ID = snapShot.Ref.ID
		subjects = append(subjects, subject)
	}

	return subjects, nil
}

func (repo *lockRepository) insert(v interface{}, subject *model.Lock) (string, error) {
	var (
		dr  = repo.GetCollection().NewDoc()
		err error
	)

	switch x := v.(type) {
	case *firestore.Transaction:
		err = x.Set(dr, subject)
	case context.Context:
		_, err = dr.Set(x, subject)
	default:
		return "", xerrors.Errorf("invalid type: %v", v)
	}

	if err != nil {
		return "", xerrors.Errorf("error in Set method: %w", err)
	}

	subject.ID = dr.ID

	return dr.ID, nil
}

func (repo *lockRepository) update(v interface{}, subject *model.Lock) error {
	var (
		dr  = repo.GetDocRef(subject.ID)
		err error
	)

	switch x := v.(type) {
	case *firestore.Transaction:
		err = x.Set(dr, subject)
	case context.Context:
		_, err = dr.Set(x, subject)
	default:
		return xerrors.Errorf("invalid type: %v", v)
	}

	if err != nil {
		return xerrors.Errorf("error in Set method: %w", err)
	}

	return nil
}

func (repo *lockRepository) strictUpdate(v interface{}, id string, param *LockUpdateParam, opts ...firestore.Precondition) error {
	var (
		dr  = repo.GetDocRef(id)
		err error
	)

	updates := updater(model.Lock{}, param)

	switch x := v.(type) {
	case *firestore.Transaction:
		err = x.Update(dr, updates, opts...)
	case context.Context:
		_, err = dr.Update(x, updates, opts...)
	default:
		return xerrors.Errorf("invalid type: %v", v)
	}

	if err != nil {
		return xerrors.Errorf("error in Update method: %w", err)
	}

	return nil
}

func (repo *lockRepository) deleteByID(v interface{}, id string) error {
	dr := repo.GetDocRef(id)
	var err error

	switch x := v.(type) {
	case *firestore.Transaction:
		err = x.Delete(dr, firestore.Exists)
	case context.Context:
		_, err = dr.Delete(x, firestore.Exists)
	default:
		return xerrors.Errorf("invalid type: %v", v)
	}

	if err != nil {
		return xerrors.Errorf("error in Delete method: %w", err)
	}

	return nil
}

func (repo *lockRepository) runQuery(v interface{}, query firestore.Query) ([]*model.Lock, error) {
	var iter *firestore.DocumentIterator

	switch x := v.(type) {
	case *firestore.Transaction:
		iter = x.Documents(query)
	case context.Context:
		iter = query.Documents(x)
	default:
		return nil, xerrors.Errorf("invalid type: %v", v)
	}

	defer iter.Stop()

	subjects := make([]*model.Lock, 0)

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, xerrors.Errorf("error in Next method: %w", err)
		}

		subject := new(model.Lock)

		if err := doc.DataTo(&subject); err != nil {
			return nil, xerrors.Errorf("error in DataTo method: %w", err)
		}

		subject.ID = doc.Ref.ID
		subjects = append(subjects, subject)
	}

	return subjects, nil
}

var lockRepositoryMeta = tagMap(model.Lock{})

// BUG(54m): there may be potential bugs
func (repo *lockRepository) list(v interface{}, req *LockListReq, q *firestore.Query) ([]*model.Lock, error) {
	if (req == nil && q == nil) || (req != nil && q != nil) {
		return nil, xerrors.New("either one should be nil")
	}

	query := func() firestore.Query {
		if q != nil {
			return *q
		}
		return repo.GetCollection().Query
	}()

	if q == nil {
		if req.Text != nil {
			for _, chain := range req.Text.QueryGroup {
				query = query.Where("text", chain.Operator, chain.Value)
			}
		}
		if req.Flag != nil {
			for _, chain := range req.Flag.QueryGroup {
				items, ok := chain.Value.(map[string]float64)
				if !ok {
					continue
				}
				for key, value := range items {
					query = query.WherePath(firestore.FieldPath{"flag", key}, chain.Operator, value)
				}
			}
		}
		if req.CreatedAt != nil {
			for _, chain := range req.CreatedAt.QueryGroup {
				query = query.Where(lockRepositoryMeta["CreatedAt"], chain.Operator, chain.Value)
			}
		}
		if req.CreatedBy != nil {
			for _, chain := range req.CreatedBy.QueryGroup {
				query = query.Where(lockRepositoryMeta["CreatedBy"], chain.Operator, chain.Value)
			}
		}
		if req.DeletedAt != nil {
			for _, chain := range req.DeletedAt.QueryGroup {
				query = query.Where(lockRepositoryMeta["DeletedAt"], chain.Operator, chain.Value)
			}
		}
		if req.DeletedBy != nil {
			for _, chain := range req.DeletedBy.QueryGroup {
				query = query.Where(lockRepositoryMeta["DeletedBy"], chain.Operator, chain.Value)
			}
		}
		if req.UpdatedAt != nil {
			for _, chain := range req.UpdatedAt.QueryGroup {
				query = query.Where(lockRepositoryMeta["UpdatedAt"], chain.Operator, chain.Value)
			}
		}
		if req.UpdatedBy != nil {
			for _, chain := range req.UpdatedBy.QueryGroup {
				query = query.Where(lockRepositoryMeta["UpdatedBy"], chain.Operator, chain.Value)
			}
		}
		if req.Version != nil {
			for _, chain := range req.Version.QueryGroup {
				query = query.Where(lockRepositoryMeta["Version"], chain.Operator, chain.Value)
			}
		}

		if !req.IncludeSoftDeleted {
			query = query.Where("DeletedAt", OpTypeEqual, nil)
		}
	}

	return repo.runQuery(v, query)
}
