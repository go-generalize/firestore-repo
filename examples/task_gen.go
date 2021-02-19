// Code generated by firestore-repo. DO NOT EDIT.
// generated version: 0.9.1
package examples

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/go-utils/xim"
	"golang.org/x/xerrors"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//go:generate mockgen -source task_gen.go -destination mock/mock_task_gen/mock_task_gen.go

// TaskRepository - Repository of Task
type TaskRepository interface {
	// Single
	Get(ctx context.Context, id string, opts ...GetOption) (*Task, error)
	GetWithDoc(ctx context.Context, doc *firestore.DocumentRef, opts ...GetOption) (*Task, error)
	Insert(ctx context.Context, subject *Task) (_ string, err error)
	Update(ctx context.Context, subject *Task) (err error)
	StrictUpdate(ctx context.Context, id string, param *TaskUpdateParam, opts ...firestore.Precondition) error
	Delete(ctx context.Context, subject *Task, opts ...DeleteOption) (err error)
	DeleteByID(ctx context.Context, id string, opts ...DeleteOption) (err error)
	// Multiple
	GetMulti(ctx context.Context, ids []string, opts ...GetOption) ([]*Task, error)
	InsertMulti(ctx context.Context, subjects []*Task) (_ []string, er error)
	UpdateMulti(ctx context.Context, subjects []*Task) (er error)
	DeleteMulti(ctx context.Context, subjects []*Task, opts ...DeleteOption) (er error)
	DeleteMultiByIDs(ctx context.Context, ids []string, opts ...DeleteOption) (er error)
	// Single(Transaction)
	GetWithTx(tx *firestore.Transaction, id string, opts ...GetOption) (*Task, error)
	GetWithDocWithTx(tx *firestore.Transaction, doc *firestore.DocumentRef, opts ...GetOption) (*Task, error)
	InsertWithTx(ctx context.Context, tx *firestore.Transaction, subject *Task) (_ string, err error)
	UpdateWithTx(ctx context.Context, tx *firestore.Transaction, subject *Task) (err error)
	StrictUpdateWithTx(tx *firestore.Transaction, id string, param *TaskUpdateParam, opts ...firestore.Precondition) error
	DeleteWithTx(ctx context.Context, tx *firestore.Transaction, subject *Task, opts ...DeleteOption) (err error)
	DeleteByIDWithTx(ctx context.Context, tx *firestore.Transaction, id string, opts ...DeleteOption) (err error)
	// Multiple(Transaction)
	GetMultiWithTx(tx *firestore.Transaction, ids []string, opts ...GetOption) ([]*Task, error)
	InsertMultiWithTx(ctx context.Context, tx *firestore.Transaction, subjects []*Task) (_ []string, er error)
	UpdateMultiWithTx(ctx context.Context, tx *firestore.Transaction, subjects []*Task) (er error)
	DeleteMultiWithTx(ctx context.Context, tx *firestore.Transaction, subjects []*Task, opts ...DeleteOption) (er error)
	DeleteMultiByIDsWithTx(ctx context.Context, tx *firestore.Transaction, ids []string, opts ...DeleteOption) (er error)
	// List
	List(ctx context.Context, req *TaskListReq, q *firestore.Query) ([]*Task, error)
	ListWithTx(tx *firestore.Transaction, req *TaskListReq, q *firestore.Query) ([]*Task, error)
	// misc
	GetCollection() *firestore.CollectionRef
	GetCollectionName() string
	GetDocRef(id string) *firestore.DocumentRef
}

// TaskRepositoryMiddleware - middleware of TaskRepository
type TaskRepositoryMiddleware interface {
	BeforeInsert(ctx context.Context, subject *Task) (bool, error)
	BeforeUpdate(ctx context.Context, old, subject *Task) (bool, error)
	BeforeDelete(ctx context.Context, subject *Task, opts ...DeleteOption) (bool, error)
	BeforeDeleteByID(ctx context.Context, ids []string, opts ...DeleteOption) (bool, error)
}

type taskRepository struct {
	collectionName   string
	firestoreClient  *firestore.Client
	middleware       []TaskRepositoryMiddleware
	uniqueRepository *uniqueRepository
}

// NewTaskRepository - constructor
func NewTaskRepository(firestoreClient *firestore.Client, middleware ...TaskRepositoryMiddleware) TaskRepository {
	return &taskRepository{
		collectionName:   "Task",
		firestoreClient:  firestoreClient,
		middleware:       middleware,
		uniqueRepository: newUniqueRepository(firestoreClient, "Task"),
	}
}

func (repo *taskRepository) beforeInsert(ctx context.Context, subject *Task) (RollbackFunc, error) {
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

func (repo *taskRepository) beforeUpdate(ctx context.Context, old, subject *Task) (RollbackFunc, error) {
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

func (repo *taskRepository) beforeDelete(ctx context.Context, subject *Task, opts ...DeleteOption) (RollbackFunc, error) {
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
func (repo *taskRepository) GetCollection() *firestore.CollectionRef {
	return repo.firestoreClient.Collection(repo.collectionName)
}

// GetCollectionName - CollectionName getter
func (repo *taskRepository) GetCollectionName() string {
	return repo.collectionName
}

// GetDocRef  - *firestore.DocumentRef getter
func (repo *taskRepository) GetDocRef(id string) *firestore.DocumentRef {
	return repo.GetCollection().Doc(id)
}

func (repo *taskRepository) saveIndexes(subject *Task) error {
	idx := xim.NewIndexes(&xim.Config{
		IgnoreCase:         true,
		SaveNoFiltersIndex: true,
	})
	{
		idx.Add(TaskIndexLabelDescEqual, subject.Desc)
		idx.AddBiunigrams(TaskIndexLabelDescLike, subject.Desc)
		idx.AddPrefixes(TaskIndexLabelDescPrefix, subject.Desc)
		idx.AddSuffixes(TaskIndexLabelDescSuffix, subject.Desc)
		idx.AddSomething(TaskIndexLabelProportionPrefix, subject.Proportion)
		idx.AddSomething(TaskIndexLabelProportionSuffix, subject.Proportion)
		idx.AddSomething(TaskIndexLabelProportionLike, subject.Proportion)
		idx.AddSomething(TaskIndexLabelProportionEqual, subject.Proportion)
	}
	indexes, err := idx.Build()
	if err != nil {
		return xerrors.Errorf("failed to index build: %w", err)
	} else if len(indexes) == 0 {
		return nil
	}

	subject.Indexes = indexes

	return nil
}

// TaskListReq - params for search
type TaskListReq struct {
	Desc       *QueryChainer
	Created    *QueryChainer
	Done       *QueryChainer
	Done2      *QueryChainer
	Count      *QueryChainer
	Count64    *QueryChainer
	NameList   *QueryChainer
	Proportion *QueryChainer
	Flag       *QueryChainer
}

// TaskUpdateParam - params for strict updates
type TaskUpdateParam struct {
	Desc       interface{}
	Created    interface{}
	Done       interface{}
	Done2      interface{}
	Count      interface{}
	Count64    interface{}
	NameList   interface{}
	Proportion interface{}
	Flag       interface{}
}

// List - search documents
// The third argument is firestore.Query, basically you can pass nil
func (repo *taskRepository) List(ctx context.Context, req *TaskListReq, q *firestore.Query) ([]*Task, error) {
	return repo.list(ctx, req, q)
}

// Get - get `Task` by `Task.ID`
func (repo *taskRepository) Get(ctx context.Context, id string, opts ...GetOption) (*Task, error) {
	doc := repo.GetDocRef(id)
	return repo.get(ctx, doc, opts...)
}

// GetWithDoc - get `Task` by *firestore.DocumentRef
func (repo *taskRepository) GetWithDoc(ctx context.Context, doc *firestore.DocumentRef, opts ...GetOption) (*Task, error) {
	return repo.get(ctx, doc, opts...)
}

// Insert - insert of `Task`
func (repo *taskRepository) Insert(ctx context.Context, subject *Task) (_ string, err error) {
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

	if err = repo.saveIndexes(subject); err != nil {
		return "", xerrors.Errorf("failed to saveIndexes: %w", err)
	}

	return repo.insert(ctx, subject)
}

// Update - update of `Task`
func (repo *taskRepository) Update(ctx context.Context, subject *Task) (err error) {
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

	if err := repo.saveIndexes(subject); err != nil {
		return xerrors.Errorf("failed to saveIndexes: %w", err)
	}

	return repo.update(ctx, subject)
}

// StrictUpdate - strict update of `Task`
func (repo *taskRepository) StrictUpdate(ctx context.Context, id string, param *TaskUpdateParam, opts ...firestore.Precondition) error {
	return repo.strictUpdate(ctx, id, param, opts...)
}

// Delete - delete of `Task`
func (repo *taskRepository) Delete(ctx context.Context, subject *Task, opts ...DeleteOption) (err error) {
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

// DeleteByID - delete `Task` by `Task.ID`
func (repo *taskRepository) DeleteByID(ctx context.Context, id string, opts ...DeleteOption) (err error) {
	subject, err := repo.Get(ctx, id)
	if err != nil {
		return xerrors.Errorf("error in Get method: %w", err)
	}

	return repo.Delete(ctx, subject, opts...)
}

// GetMulti - get `Task` in bulk by array of `Task.ID`
func (repo *taskRepository) GetMulti(ctx context.Context, ids []string, opts ...GetOption) ([]*Task, error) {
	return repo.getMulti(ctx, ids, opts...)
}

// InsertMulti - bulk insert of `Task`
func (repo *taskRepository) InsertMulti(ctx context.Context, subjects []*Task) (_ []string, er error) {
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

		if err := repo.saveIndexes(subjects[i]); err != nil {
			return nil, xerrors.Errorf("failed to saveIndexes: %w", err)
		}

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

// UpdateMulti - bulk update of `Task`
func (repo *taskRepository) UpdateMulti(ctx context.Context, subjects []*Task) (er error) {
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

		old := new(Task)
		if err = snapShot.DataTo(&old); err != nil {
			return xerrors.Errorf("error in DataTo method: %w", err)
		}

		rb, err := repo.beforeUpdate(ctx, old, subject)
		if err != nil {
			return xerrors.Errorf("before update error(%d) [%v]: %w", i, subject.ID, err)
		}
		rbs = append(rbs, rb)

		if err := repo.saveIndexes(subjects[i]); err != nil {
			return xerrors.Errorf("failed to saveIndexes: %w", err)
		}

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

// DeleteMulti - bulk delete of `Task`
func (repo *taskRepository) DeleteMulti(ctx context.Context, subjects []*Task, opts ...DeleteOption) (er error) {
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

// DeleteMultiByIDs - delete `Task` in bulk by array of `Task.ID`
func (repo *taskRepository) DeleteMultiByIDs(ctx context.Context, ids []string, opts ...DeleteOption) (er error) {
	subjects := make([]*Task, len(ids))

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

func (repo *taskRepository) ListWithTx(tx *firestore.Transaction, req *TaskListReq, q *firestore.Query) ([]*Task, error) {
	return repo.list(tx, req, q)
}

// GetWithTx - get `Task` by `Task.ID` in transaction
func (repo *taskRepository) GetWithTx(tx *firestore.Transaction, id string, opts ...GetOption) (*Task, error) {
	doc := repo.GetDocRef(id)
	return repo.get(tx, doc, opts...)
}

// GetWithDocWithTx - get `Task` by *firestore.DocumentRef in transaction
func (repo *taskRepository) GetWithDocWithTx(tx *firestore.Transaction, doc *firestore.DocumentRef, opts ...GetOption) (*Task, error) {
	return repo.get(tx, doc, opts...)
}

// InsertWithTx - insert of `Task` in transaction
func (repo *taskRepository) InsertWithTx(ctx context.Context, tx *firestore.Transaction, subject *Task) (_ string, err error) {
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

	if err := repo.saveIndexes(subject); err != nil {
		return "", xerrors.Errorf("failed to saveIndexes: %w", err)
	}

	return repo.insert(tx, subject)
}

// UpdateWithTx - update of `Task` in transaction
func (repo *taskRepository) UpdateWithTx(ctx context.Context, tx *firestore.Transaction, subject *Task) (err error) {
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

	if err := repo.saveIndexes(subject); err != nil {
		return xerrors.Errorf("failed to saveIndexes: %w", err)
	}

	return repo.update(tx, subject)
}

// StrictUpdateWithTx - strict update of `Task` in transaction
func (repo *taskRepository) StrictUpdateWithTx(tx *firestore.Transaction, id string, param *TaskUpdateParam, opts ...firestore.Precondition) error {
	return repo.strictUpdate(tx, id, param, opts...)
}

// DeleteWithTx - delete of `Task` in transaction
func (repo *taskRepository) DeleteWithTx(ctx context.Context, tx *firestore.Transaction, subject *Task, opts ...DeleteOption) (err error) {
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

// DeleteByIDWithTx - delete `Task` by `Task.ID` in transaction
func (repo *taskRepository) DeleteByIDWithTx(ctx context.Context, tx *firestore.Transaction, id string, opts ...DeleteOption) (err error) {
	subject, err := repo.GetWithTx(tx, id)
	if err != nil {
		return xerrors.Errorf("error in GetWithTx method: %w", err)
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

// GetMultiWithTx - get `Task` in bulk by array of `Task.ID` in transaction
func (repo *taskRepository) GetMultiWithTx(tx *firestore.Transaction, ids []string, opts ...GetOption) ([]*Task, error) {
	return repo.getMulti(tx, ids, opts...)
}

// InsertMultiWithTx - bulk insert of `Task` in transaction
func (repo *taskRepository) InsertMultiWithTx(ctx context.Context, tx *firestore.Transaction, subjects []*Task) (_ []string, er error) {
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

		if err := repo.saveIndexes(subjects[i]); err != nil {
			return nil, xerrors.Errorf("failed to saveIndexes: %w", err)
		}

		id, err := repo.insert(tx, subjects[i])
		if err != nil {
			return nil, xerrors.Errorf("error in insert method(%d) [%v]: %w", i, subjects[i].ID, err)
		}
		ids[i] = id
	}

	return ids, nil
}

// UpdateMultiWithTx - bulk update of `Task` in transaction
func (repo *taskRepository) UpdateMultiWithTx(ctx context.Context, tx *firestore.Transaction, subjects []*Task) (er error) {
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
		if err := repo.saveIndexes(subjects[i]); err != nil {
			return xerrors.Errorf("failed to saveIndexes: %w", err)
		}
		if err := repo.update(tx, subjects[i]); err != nil {
			return xerrors.Errorf("error in update method(%d) [%v]: %w", i, subjects[i].ID, err)
		}
	}

	return nil
}

// DeleteMultiWithTx - bulk delete of `Task` in transaction
func (repo *taskRepository) DeleteMultiWithTx(ctx context.Context, tx *firestore.Transaction, subjects []*Task, opts ...DeleteOption) (er error) {
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
	}

	for i := range subjects {
		if err := repo.deleteByID(tx, subjects[i].ID); err != nil {
			return xerrors.Errorf("error in delete method(%d) [%v]: %w", i, subjects[i].ID, err)
		}
	}

	return nil
}

// DeleteMultiByIDWithTx - delete `Task` in bulk by array of `Task.ID` in transaction
func (repo *taskRepository) DeleteMultiByIDsWithTx(ctx context.Context, tx *firestore.Transaction, ids []string, opts ...DeleteOption) (er error) {
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
	}

	for i := range ids {
		if err := repo.deleteByID(tx, ids[i]); err != nil {
			return xerrors.Errorf("error in delete method(%d) [%v]: %w", i, ids[i], err)
		}
	}

	return nil
}

func (repo *taskRepository) get(v interface{}, doc *firestore.DocumentRef, _ ...GetOption) (*Task, error) {
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

	subject := new(Task)
	if err := snapShot.DataTo(&subject); err != nil {
		return nil, xerrors.Errorf("error in DataTo method: %w", err)
	}

	subject.ID = snapShot.Ref.ID

	return subject, nil
}

func (repo *taskRepository) getMulti(v interface{}, ids []string, _ ...GetOption) ([]*Task, error) {
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

	subjects := make([]*Task, 0, len(ids))
	for _, snapShot := range snapShots {
		subject := new(Task)
		if err := snapShot.DataTo(&subject); err != nil {
			return nil, xerrors.Errorf("error in DataTo method: %w", err)
		}

		subject.ID = snapShot.Ref.ID
		subjects = append(subjects, subject)
	}

	return subjects, nil
}

func (repo *taskRepository) insert(v interface{}, subject *Task) (string, error) {
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

func (repo *taskRepository) update(v interface{}, subject *Task) error {
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

func (repo *taskRepository) strictUpdate(v interface{}, id string, param *TaskUpdateParam, opts ...firestore.Precondition) error {
	var (
		dr  = repo.GetDocRef(id)
		err error
	)

	updates := updater(Task{}, param)

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

func (repo *taskRepository) deleteByID(v interface{}, id string) error {
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

func (repo *taskRepository) runQuery(v interface{}, query firestore.Query) ([]*Task, error) {
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

	subjects := make([]*Task, 0)

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, xerrors.Errorf("error in Next method: %w", err)
		}

		subject := new(Task)

		if err := doc.DataTo(&subject); err != nil {
			return nil, xerrors.Errorf("error in DataTo method: %w", err)
		}

		subject.ID = doc.Ref.ID
		subjects = append(subjects, subject)
	}

	return subjects, nil
}

// BUG(54m): there may be potential bugs
func (repo *taskRepository) list(v interface{}, req *TaskListReq, q *firestore.Query) ([]*Task, error) {
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
		filters := xim.NewFilters(&xim.Config{
			IgnoreCase:         true,
			SaveNoFiltersIndex: true,
		})

		if req.Desc != nil {
			for _, chain := range req.Desc.QueryGroup {
				query = query.Where("description", chain.Operator, chain.Value)
			}
			value, ok := req.Desc.Filter.Value.(string)
			// The value of the "indexer" tag = "e,p,s,l"
			// Add/AddBiunigrams/AddPrefix/AddSuffix is valid.
			for _, filter := range req.Desc.Filter.FilterTypes {
				switch filter {
				case FilterTypeAddBiunigrams:
					if ok {
						filters.AddBiunigrams(TaskIndexLabelDescLike, value)
					}
				case FilterTypeAddPrefix:
					if ok {
						filters.AddPrefix(TaskIndexLabelDescPrefix, value)
					}
				case FilterTypeAddSuffix:
					if ok {
						filters.AddSuffix(TaskIndexLabelDescSuffix, value)
					}
				// Treat `Add` or otherwise as `Equal`.
				case FilterTypeAdd:
					fallthrough
				default:
					if !ok {
						filters.AddSomething(TaskIndexLabelDescEqual, req.Desc.Filter.Value)
						continue
					}
					filters.Add(TaskIndexLabelDescEqual, value)
				}
			}
		}
		if req.Created != nil {
			for _, chain := range req.Created.QueryGroup {
				query = query.Where("created", chain.Operator, chain.Value)
			}
			value, ok := req.Created.Filter.Value.(string)
			for _, filter := range req.Created.Filter.FilterTypes {
				switch filter {
				// Treat `Add` or otherwise as `Equal`.
				case FilterTypeAdd:
					fallthrough
				default:
					if !ok {
						filters.AddSomething(TaskIndexLabelCreatedEqual, req.Created.Filter.Value)
						continue
					}
					filters.Add(TaskIndexLabelCreatedEqual, value)
				}
			}
		}
		if req.Done != nil {
			for _, chain := range req.Done.QueryGroup {
				query = query.Where("done", chain.Operator, chain.Value)
			}
			value, ok := req.Done.Filter.Value.(string)
			for _, filter := range req.Done.Filter.FilterTypes {
				switch filter {
				// Treat `Add` or otherwise as `Equal`.
				case FilterTypeAdd:
					fallthrough
				default:
					if !ok {
						filters.AddSomething(TaskIndexLabelDoneEqual, req.Done.Filter.Value)
						continue
					}
					filters.Add(TaskIndexLabelDoneEqual, value)
				}
			}
		}
		if req.Done2 != nil {
			for _, chain := range req.Done2.QueryGroup {
				query = query.Where("done2", chain.Operator, chain.Value)
			}
			value, ok := req.Done2.Filter.Value.(string)
			for _, filter := range req.Done2.Filter.FilterTypes {
				switch filter {
				// Treat `Add` or otherwise as `Equal`.
				case FilterTypeAdd:
					fallthrough
				default:
					if !ok {
						filters.AddSomething(TaskIndexLabelDone2Equal, req.Done2.Filter.Value)
						continue
					}
					filters.Add(TaskIndexLabelDone2Equal, value)
				}
			}
		}
		if req.Count != nil {
			for _, chain := range req.Count.QueryGroup {
				query = query.Where("count", chain.Operator, chain.Value)
			}
			value, ok := req.Count.Filter.Value.(string)
			for _, filter := range req.Count.Filter.FilterTypes {
				switch filter {
				// Treat `Add` or otherwise as `Equal`.
				case FilterTypeAdd:
					fallthrough
				default:
					if !ok {
						filters.AddSomething(TaskIndexLabelCountEqual, req.Count.Filter.Value)
						continue
					}
					filters.Add(TaskIndexLabelCountEqual, value)
				}
			}
		}
		if req.Count64 != nil {
			for _, chain := range req.Count64.QueryGroup {
				query = query.Where("count64", chain.Operator, chain.Value)
			}
			value, ok := req.Count64.Filter.Value.(string)
			for _, filter := range req.Count64.Filter.FilterTypes {
				switch filter {
				// Treat `Add` or otherwise as `Equal`.
				case FilterTypeAdd:
					fallthrough
				default:
					if !ok {
						filters.AddSomething(TaskIndexLabelCount64Equal, req.Count64.Filter.Value)
						continue
					}
					filters.Add(TaskIndexLabelCount64Equal, value)
				}
			}
		}
		if req.NameList != nil {
			for _, chain := range req.NameList.QueryGroup {
				query = query.Where("nameList", chain.Operator, chain.Value)
			}
		}
		if req.Proportion != nil {
			for _, chain := range req.Proportion.QueryGroup {
				query = query.Where("proportion", chain.Operator, chain.Value)
			}
			value, ok := req.Proportion.Filter.Value.(string)
			// The value of the "indexer" tag = "e"
			for _, filter := range req.Proportion.Filter.FilterTypes {
				switch filter {
				// Treat `Add` or otherwise as `Equal`.
				case FilterTypeAdd:
					fallthrough
				default:
					if !ok {
						filters.AddSomething(TaskIndexLabelProportionEqual, req.Proportion.Filter.Value)
						continue
					}
					filters.Add(TaskIndexLabelProportionEqual, value)
				}
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

		build, err := filters.Build()
		if err != nil {
			return nil, xerrors.Errorf("failed to filter build: %w", err)
		}
		for key := range build {
			query = query.WherePath(firestore.FieldPath{"indexes", key}, OpTypeEqual, true)
		}
	}

	return repo.runQuery(v, query)
}
