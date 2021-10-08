// Code generated by firestore-repo. DO NOT EDIT.
// generated version: 1.10.0
package examples

import (
	"context"

	"cloud.google.com/go/firestore"
	"golang.org/x/xerrors"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//go:generate mockgen -source $GOFILE -destination mock/mock_history_gen/mock_history_gen.go

// HistoryRepository - Repository of History
type HistoryRepository interface {
	// Single
	Get(ctx context.Context, id string, opts ...GetOption) (*History, error)
	GetWithDoc(ctx context.Context, doc *firestore.DocumentRef, opts ...GetOption) (*History, error)
	Insert(ctx context.Context, subject *History) (_ string, err error)
	Update(ctx context.Context, subject *History) (err error)
	StrictUpdate(ctx context.Context, id string, param *HistoryUpdateParam, opts ...firestore.Precondition) error
	Delete(ctx context.Context, subject *History, opts ...DeleteOption) (err error)
	DeleteByID(ctx context.Context, id string, opts ...DeleteOption) (err error)
	// Multiple
	GetMulti(ctx context.Context, ids []string, opts ...GetOption) ([]*History, error)
	InsertMulti(ctx context.Context, subjects []*History) (_ []string, er error)
	UpdateMulti(ctx context.Context, subjects []*History) (er error)
	DeleteMulti(ctx context.Context, subjects []*History, opts ...DeleteOption) (er error)
	DeleteMultiByIDs(ctx context.Context, ids []string, opts ...DeleteOption) (er error)
	// Single(Transaction)
	GetWithTx(tx *firestore.Transaction, id string, opts ...GetOption) (*History, error)
	GetWithDocWithTx(tx *firestore.Transaction, doc *firestore.DocumentRef, opts ...GetOption) (*History, error)
	InsertWithTx(ctx context.Context, tx *firestore.Transaction, subject *History) (_ string, err error)
	UpdateWithTx(ctx context.Context, tx *firestore.Transaction, subject *History) (err error)
	StrictUpdateWithTx(tx *firestore.Transaction, id string, param *HistoryUpdateParam, opts ...firestore.Precondition) error
	DeleteWithTx(ctx context.Context, tx *firestore.Transaction, subject *History, opts ...DeleteOption) (err error)
	DeleteByIDWithTx(ctx context.Context, tx *firestore.Transaction, id string, opts ...DeleteOption) (err error)
	// Multiple(Transaction)
	GetMultiWithTx(tx *firestore.Transaction, ids []string, opts ...GetOption) ([]*History, error)
	InsertMultiWithTx(ctx context.Context, tx *firestore.Transaction, subjects []*History) (_ []string, er error)
	UpdateMultiWithTx(ctx context.Context, tx *firestore.Transaction, subjects []*History) (er error)
	DeleteMultiWithTx(ctx context.Context, tx *firestore.Transaction, subjects []*History, opts ...DeleteOption) (er error)
	DeleteMultiByIDsWithTx(ctx context.Context, tx *firestore.Transaction, ids []string, opts ...DeleteOption) (er error)
	// Search
	Search(ctx context.Context, param *HistorySearchParam, q *firestore.Query) ([]*History, error)
	SearchWithTx(tx *firestore.Transaction, param *HistorySearchParam, q *firestore.Query) ([]*History, error)
	// misc
	GetCollection() *firestore.CollectionRef
	GetCollectionName() string
	GetDocRef(id string) *firestore.DocumentRef
	RunInTransaction() func(ctx context.Context, f func(context.Context, *firestore.Transaction) error, opts ...firestore.TransactionOption) (err error)
	SetParentDoc(doc *firestore.DocumentRef)
	Free()
}

// HistoryRepositoryMiddleware - middleware of HistoryRepository
type HistoryRepositoryMiddleware interface {
	BeforeInsert(ctx context.Context, subject *History) (bool, error)
	BeforeUpdate(ctx context.Context, old, subject *History) (bool, error)
	BeforeDelete(ctx context.Context, subject *History, opts ...DeleteOption) (bool, error)
	BeforeDeleteByID(ctx context.Context, ids []string, opts ...DeleteOption) (bool, error)
}

type historyRepository struct {
	collectionName   string
	firestoreClient  *firestore.Client
	parentDocument   *firestore.DocumentRef
	collectionGroup  *firestore.CollectionGroupRef
	middleware       []HistoryRepositoryMiddleware
	uniqueRepository *uniqueRepository
}

// NewHistoryRepository - constructor
func NewHistoryRepository(firestoreClient *firestore.Client, parentDocument *firestore.DocumentRef, middleware ...HistoryRepositoryMiddleware) HistoryRepository {
	return &historyRepository{
		collectionName:   "History",
		firestoreClient:  firestoreClient,
		parentDocument:   parentDocument,
		middleware:       middleware,
		uniqueRepository: newUniqueRepository(firestoreClient, "History"),
	}
}

// NewHistoryCollectionGroupRepository - constructor
func NewHistoryCollectionGroupRepository(firestoreClient *firestore.Client) HistoryRepository {
	return &historyRepository{
		collectionName:  "History",
		collectionGroup: firestoreClient.CollectionGroup("History"),
	}
}

func (repo *historyRepository) beforeInsert(ctx context.Context, subject *History) (RollbackFunc, error) {
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

func (repo *historyRepository) beforeUpdate(ctx context.Context, old, subject *History) (RollbackFunc, error) {
	if ctx.Value(transactionInProgressKey{}) != nil && old == nil {
		var err error
		doc := repo.GetDocRef(subject.ID)
		old, err = repo.get(context.Background(), doc)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return nil, ErrNotFound
			}
			return nil, xerrors.Errorf("error in Get method: %w", err)
		}
	}
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

func (repo *historyRepository) beforeDelete(ctx context.Context, subject *History, opts ...DeleteOption) (RollbackFunc, error) {
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
func (repo *historyRepository) GetCollection() *firestore.CollectionRef {
	if repo.collectionGroup != nil {
		return nil
	}
	return repo.parentDocument.Collection(repo.collectionName)
}

// GetCollectionName - CollectionName getter
func (repo *historyRepository) GetCollectionName() string {
	return repo.collectionName
}

// GetDocRef - *firestore.DocumentRef getter
func (repo *historyRepository) GetDocRef(id string) *firestore.DocumentRef {
	if repo.collectionGroup != nil {
		return nil
	}
	return repo.GetCollection().Doc(id)
}

// RunInTransaction - (*firestore.Client).RunTransaction getter
func (repo *historyRepository) RunInTransaction() func(ctx context.Context, f func(context.Context, *firestore.Transaction) error, opts ...firestore.TransactionOption) (err error) {
	return repo.firestoreClient.RunTransaction
}

// SetParentDoc - parent document setter
func (repo *historyRepository) SetParentDoc(doc *firestore.DocumentRef) {
	if doc == nil {
		return
	}
	repo.parentDocument = doc
}

// Free - parent document releaser
func (repo *historyRepository) Free() {
	repo.parentDocument = nil
}

// HistorySearchParam - params for search
type HistorySearchParam struct {
	IsSubCollection *QueryChainer

	CursorLimit int
}

// HistoryUpdateParam - params for strict updates
type HistoryUpdateParam struct {
	IsSubCollection interface{}
}

// Search - search documents
// The third argument is firestore.Query, basically you can pass nil
func (repo *historyRepository) Search(ctx context.Context, param *HistorySearchParam, q *firestore.Query) ([]*History, error) {
	return repo.search(ctx, param, q)
}

// Get - get `History` by `History.ID`
func (repo *historyRepository) Get(ctx context.Context, id string, opts ...GetOption) (*History, error) {
	if repo.collectionGroup != nil {
		return nil, ErrNotAvailableCG
	}
	doc := repo.GetDocRef(id)
	return repo.get(ctx, doc, opts...)
}

// GetWithDoc - get `History` by *firestore.DocumentRef
func (repo *historyRepository) GetWithDoc(ctx context.Context, doc *firestore.DocumentRef, opts ...GetOption) (*History, error) {
	if repo.collectionGroup != nil {
		return nil, ErrNotAvailableCG
	}
	return repo.get(ctx, doc, opts...)
}

// Insert - insert of `History`
func (repo *historyRepository) Insert(ctx context.Context, subject *History) (_ string, err error) {
	if repo.collectionGroup != nil {
		return "", ErrNotAvailableCG
	}
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

// Update - update of `History`
func (repo *historyRepository) Update(ctx context.Context, subject *History) (err error) {
	if repo.collectionGroup != nil {
		return ErrNotAvailableCG
	}
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

// StrictUpdate - strict update of `History`
func (repo *historyRepository) StrictUpdate(ctx context.Context, id string, param *HistoryUpdateParam, opts ...firestore.Precondition) error {
	if repo.collectionGroup != nil {
		return ErrNotAvailableCG
	}
	return repo.strictUpdate(ctx, id, param, opts...)
}

// Delete - delete of `History`
func (repo *historyRepository) Delete(ctx context.Context, subject *History, opts ...DeleteOption) (err error) {
	if repo.collectionGroup != nil {
		return ErrNotAvailableCG
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

// DeleteByID - delete `History` by `History.ID`
func (repo *historyRepository) DeleteByID(ctx context.Context, id string, opts ...DeleteOption) (err error) {
	if repo.collectionGroup != nil {
		return ErrNotAvailableCG
	}
	subject, err := repo.Get(ctx, id)
	if err != nil {
		return xerrors.Errorf("error in Get method: %w", err)
	}

	return repo.Delete(ctx, subject, opts...)
}

// GetMulti - get `History` in bulk by array of `History.ID`
func (repo *historyRepository) GetMulti(ctx context.Context, ids []string, opts ...GetOption) ([]*History, error) {
	if repo.collectionGroup != nil {
		return nil, ErrNotAvailableCG
	}
	return repo.getMulti(ctx, ids, opts...)
}

// InsertMulti - bulk insert of `History`
func (repo *historyRepository) InsertMulti(ctx context.Context, subjects []*History) (_ []string, er error) {
	if repo.collectionGroup != nil {
		return nil, ErrNotAvailableCG
	}
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

// UpdateMulti - bulk update of `History`
func (repo *historyRepository) UpdateMulti(ctx context.Context, subjects []*History) (er error) {
	if repo.collectionGroup != nil {
		return ErrNotAvailableCG
	}
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

		old := new(History)
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

// DeleteMulti - bulk delete of `History`
func (repo *historyRepository) DeleteMulti(ctx context.Context, subjects []*History, opts ...DeleteOption) (er error) {
	if repo.collectionGroup != nil {
		return ErrNotAvailableCG
	}
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

		batch.Delete(ref)

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

// DeleteMultiByIDs - delete `History` in bulk by array of `History.ID`
func (repo *historyRepository) DeleteMultiByIDs(ctx context.Context, ids []string, opts ...DeleteOption) (er error) {
	if repo.collectionGroup != nil {
		return ErrNotAvailableCG
	}
	subjects := make([]*History, len(ids))

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

func (repo *historyRepository) SearchWithTx(tx *firestore.Transaction, param *HistorySearchParam, q *firestore.Query) ([]*History, error) {
	return repo.search(tx, param, q)
}

// GetWithTx - get `History` by `History.ID` in transaction
func (repo *historyRepository) GetWithTx(tx *firestore.Transaction, id string, opts ...GetOption) (*History, error) {
	if repo.collectionGroup != nil {
		return nil, ErrNotAvailableCG
	}
	doc := repo.GetDocRef(id)
	return repo.get(tx, doc, opts...)
}

// GetWithDocWithTx - get `History` by *firestore.DocumentRef in transaction
func (repo *historyRepository) GetWithDocWithTx(tx *firestore.Transaction, doc *firestore.DocumentRef, opts ...GetOption) (*History, error) {
	if repo.collectionGroup != nil {
		return nil, ErrNotAvailableCG
	}
	return repo.get(tx, doc, opts...)
}

// InsertWithTx - insert of `History` in transaction
func (repo *historyRepository) InsertWithTx(ctx context.Context, tx *firestore.Transaction, subject *History) (_ string, err error) {
	if repo.collectionGroup != nil {
		return "", ErrNotAvailableCG
	}
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

// UpdateWithTx - update of `History` in transaction
func (repo *historyRepository) UpdateWithTx(ctx context.Context, tx *firestore.Transaction, subject *History) (err error) {
	if repo.collectionGroup != nil {
		return ErrNotAvailableCG
	}
	rb, err := repo.beforeUpdate(context.WithValue(ctx, transactionInProgressKey{}, 1), nil, subject)
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

// StrictUpdateWithTx - strict update of `History` in transaction
func (repo *historyRepository) StrictUpdateWithTx(tx *firestore.Transaction, id string, param *HistoryUpdateParam, opts ...firestore.Precondition) error {
	if repo.collectionGroup != nil {
		return ErrNotAvailableCG
	}
	return repo.strictUpdate(tx, id, param, opts...)
}

// DeleteWithTx - delete of `History` in transaction
func (repo *historyRepository) DeleteWithTx(ctx context.Context, tx *firestore.Transaction, subject *History, opts ...DeleteOption) (err error) {
	if repo.collectionGroup != nil {
		return ErrNotAvailableCG
	}
	rb, err := repo.beforeDelete(context.WithValue(ctx, transactionInProgressKey{}, 1), subject, opts...)
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

// DeleteByIDWithTx - delete `History` by `History.ID` in transaction
func (repo *historyRepository) DeleteByIDWithTx(ctx context.Context, tx *firestore.Transaction, id string, opts ...DeleteOption) (err error) {
	if repo.collectionGroup != nil {
		return ErrNotAvailableCG
	}
	subject, err := repo.Get(context.Background(), id)
	if err != nil {
		return xerrors.Errorf("error in Get method: %w", err)
	}

	rb, err := repo.beforeDelete(context.WithValue(ctx, transactionInProgressKey{}, 1), subject, opts...)
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

// GetMultiWithTx - get `History` in bulk by array of `History.ID` in transaction
func (repo *historyRepository) GetMultiWithTx(tx *firestore.Transaction, ids []string, opts ...GetOption) ([]*History, error) {
	if repo.collectionGroup != nil {
		return nil, ErrNotAvailableCG
	}
	return repo.getMulti(tx, ids, opts...)
}

// InsertMultiWithTx - bulk insert of `History` in transaction
func (repo *historyRepository) InsertMultiWithTx(ctx context.Context, tx *firestore.Transaction, subjects []*History) (_ []string, er error) {
	if repo.collectionGroup != nil {
		return nil, ErrNotAvailableCG
	}
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

// UpdateMultiWithTx - bulk update of `History` in transaction
func (repo *historyRepository) UpdateMultiWithTx(ctx context.Context, tx *firestore.Transaction, subjects []*History) (er error) {
	if repo.collectionGroup != nil {
		return ErrNotAvailableCG
	}
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

	ctx = context.WithValue(ctx, transactionInProgressKey{}, 1)
	for i := range subjects {
		rb, err := repo.beforeUpdate(ctx, nil, subjects[i])
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

// DeleteMultiWithTx - bulk delete of `History` in transaction
func (repo *historyRepository) DeleteMultiWithTx(ctx context.Context, tx *firestore.Transaction, subjects []*History, opts ...DeleteOption) (er error) {
	if repo.collectionGroup != nil {
		return ErrNotAvailableCG
	}
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

	var isHardDeleteMode bool
	if len(opts) > 0 {
		isHardDeleteMode = opts[0].Mode == DeleteModeHard
	}
	opt := GetOption{
		IncludeSoftDeleted: isHardDeleteMode,
	}
	for i := range subjects {
		dr := repo.GetDocRef(subjects[i].ID)
		if _, err := repo.get(context.Background(), dr, opt); err != nil {
			if status.Code(err) == codes.NotFound {
				return xerrors.Errorf("not found(%d) [%v]", i, subjects[i].ID)
			}
			return xerrors.Errorf("error in get method(%d) [%v]: %w", i, subjects[i].ID, err)
		}

		rb, err := repo.beforeDelete(context.WithValue(ctx, transactionInProgressKey{}, 1), subjects[i], opts...)
		if err != nil {
			return xerrors.Errorf("before delete error(%d) [%v]: %w", i, subjects[i].ID, err)
		}
		rbs = append(rbs, rb)
	}

	for i := range subjects {
		if err := repo.deleteByID(tx, subjects[i].ID); err != nil {
			return xerrors.Errorf("error in delete method(%d) [%v]: %w", i, subjects[i].ID, err)
		}
	}

	return nil
}

// DeleteMultiByIDWithTx - delete `History` in bulk by array of `History.ID` in transaction
func (repo *historyRepository) DeleteMultiByIDsWithTx(ctx context.Context, tx *firestore.Transaction, ids []string, opts ...DeleteOption) (er error) {
	if repo.collectionGroup != nil {
		return ErrNotAvailableCG
	}
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

	for i := range ids {
		dr := repo.GetDocRef(ids[i])
		subject, err := repo.get(context.Background(), dr)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return xerrors.Errorf("not found(%d) [%v]", i, ids[i])
			}
			return xerrors.Errorf("error in get method(%d) [%v]: %w", i, ids[i], err)
		}

		rb, err := repo.beforeDelete(context.WithValue(ctx, transactionInProgressKey{}, 1), subject, opts...)
		if err != nil {
			return xerrors.Errorf("before delete error(%d) [%v]: %w", i, subject.ID, err)
		}
		rbs = append(rbs, rb)
	}

	for i := range ids {
		if err := repo.deleteByID(tx, ids[i]); err != nil {
			return xerrors.Errorf("error in delete method(%d) [%v]: %w", i, ids[i], err)
		}
	}

	return nil
}

func (repo *historyRepository) get(v interface{}, doc *firestore.DocumentRef, _ ...GetOption) (*History, error) {
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

	subject := new(History)
	if err := snapShot.DataTo(&subject); err != nil {
		return nil, xerrors.Errorf("error in DataTo method: %w", err)
	}

	subject.ID = snapShot.Ref.ID

	return subject, nil
}

func (repo *historyRepository) getMulti(v interface{}, ids []string, _ ...GetOption) ([]*History, error) {
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

	subjects := make([]*History, 0, len(ids))
	mErr := NewMultiErrors()
	for i, snapShot := range snapShots {
		if !snapShot.Exists() {
			mErr = append(mErr, NewMultiError(i, ErrNotFound))
			continue
		}

		subject := new(History)
		if err = snapShot.DataTo(&subject); err != nil {
			return nil, xerrors.Errorf("error in DataTo method: %w", err)
		}

		subject.ID = snapShot.Ref.ID
		subjects = append(subjects, subject)
	}

	if len(mErr) == 0 {
		return subjects, nil
	}

	return subjects, mErr
}

func (repo *historyRepository) insert(v interface{}, subject *History) (string, error) {
	var (
		dr  = repo.GetCollection().NewDoc()
		err error
	)

	switch x := v.(type) {
	case *firestore.Transaction:
		err = x.Create(dr, subject)
	case context.Context:
		_, err = dr.Create(x, subject)
	default:
		return "", xerrors.Errorf("invalid type: %v", v)
	}

	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return "", xerrors.Errorf("error in Create method: err=%+v: %w", err, ErrAlreadyExists)
		}
		return "", xerrors.Errorf("error in Create method: %w", err)
	}

	subject.ID = dr.ID

	return dr.ID, nil
}

func (repo *historyRepository) update(v interface{}, subject *History) error {
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

func (repo *historyRepository) strictUpdate(v interface{}, id string, param *HistoryUpdateParam, opts ...firestore.Precondition) error {
	var (
		dr  = repo.GetDocRef(id)
		err error
	)

	updates := updater(History{}, param)

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

func (repo *historyRepository) deleteByID(v interface{}, id string) error {
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

func (repo *historyRepository) runQuery(v interface{}, query firestore.Query) ([]*History, error) {
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

	subjects := make([]*History, 0)

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, xerrors.Errorf("error in Next method: %w", err)
		}

		subject := new(History)

		if err := doc.DataTo(&subject); err != nil {
			return nil, xerrors.Errorf("error in DataTo method: %w", err)
		}

		subject.ID = doc.Ref.ID
		subjects = append(subjects, subject)
	}

	return subjects, nil
}

// BUG(54m): there may be potential bugs
func (repo *historyRepository) search(v interface{}, param *HistorySearchParam, q *firestore.Query) ([]*History, error) {
	if (param == nil && q == nil) || (param != nil && q != nil) {
		return nil, xerrors.New("either one should be nil")
	}

	query := func() firestore.Query {
		if q != nil {
			return *q
		}
		if repo.collectionGroup != nil {
			return repo.collectionGroup.Query
		}
		return repo.GetCollection().Query
	}()

	if q == nil {
		if param.IsSubCollection != nil {
			for _, chain := range param.IsSubCollection.QueryGroup {
				query = query.Where("IsSubCollection", chain.Operator, chain.Value)
			}
			if direction := param.IsSubCollection.OrderByDirection; direction > 0 {
				query = query.OrderBy("IsSubCollection", direction)
				query = param.IsSubCollection.BuildCursorQuery(query)
			}
		}

		if l := param.CursorLimit; l > 0 {
			query = query.Limit(l)
		}
	}

	return repo.runQuery(v, query)
}
