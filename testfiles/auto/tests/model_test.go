// +build internal

package tests

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	model "github.com/go-generalize/firestore-repo/testfiles/auto"
	"golang.org/x/xerrors"
)

func initFirestoreClient(t *testing.T) *firestore.Client {
	t.Helper()

	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		os.Setenv("FIRESTORE_EMULATOR_HOST", "localhost:8000")
	}

	os.Setenv("FIRESTORE_PROJECT_ID", "project-id-in-google")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := firestore.NewClient(ctx, "testing")

	if err != nil {
		t.Fatalf("failed to initialize firestore client: %+v", err)
	}

	return client
}

func compareTask(t *testing.T, expected, actual *model.Task) {
	t.Helper()

	if actual.ID != expected.ID {
		t.Fatalf("unexpected identity: %s (expected: %s)", actual.ID, expected.ID)
	}

	if !actual.Created.Equal(expected.Created) {
		t.Fatalf("unexpected time: %s(expected: %s)", actual.Created, expected.Created)
	}

	if actual.Desc != expected.Desc {
		t.Fatalf("unexpected desc: %s(expected: %s)", actual.Desc, expected.Created)
	}

	if actual.Done != expected.Done {
		t.Fatalf("unexpected done: %v(expected: %v)", actual.Done, expected.Done)
	}
}

func TestFirestore(t *testing.T) {
	client := initFirestoreClient(t)

	taskRepo := model.NewTaskRepository(client)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	var ids []string
	defer func() {
		defer cancel()
		if err := taskRepo.DeleteMultiByIDs(ctx, ids); err != nil {
			t.Fatal(err)
		}
	}()

	now := time.Unix(0, time.Now().UnixNano())
	desc := "Hello, World!"

	t.Run("Multi", func(tr *testing.T) {
		tks := make([]*model.Task, 0)
		for i := int64(1); i <= 10; i++ {
			tk := &model.Task{
				Desc:       fmt.Sprintf("%s%d", desc, i),
				Created:    now,
				Done:       true,
				Done2:      false,
				Count:      int(i),
				Count64:    0,
				Proportion: 0.12345 + float64(i),
				NameList:   []string{"a", "b", "c"},
				Flag: map[string]float64{
					"1": 1.1,
					"2": 2.2,
					"3": 3.3,
				},
			}
			tks = append(tks, tk)
		}
		idList, err := taskRepo.InsertMulti(ctx, tks)
		if err != nil {
			tr.Fatalf("%+v", err)
		}
		ids = append(ids, idList...)

		tks2 := make([]*model.Task, 0)
		for i := int64(1); i <= 10; i++ {
			tk := &model.Task{
				ID:         ids[i-1],
				Desc:       fmt.Sprintf("%s%d", desc, i),
				Created:    now,
				Done:       true,
				Done2:      false,
				Count:      int(i),
				Count64:    i,
				Proportion: 0.12345 + float64(i),
				NameList:   []string{"a", "b", "c"},
				Flag: map[string]float64{
					"4": 4.4,
					"5": 5.5,
					"6": 6.6,
				},
			}
			tks2 = append(tks2, tk)
		}
		if err := taskRepo.UpdateMulti(ctx, tks2); err != nil {
			tr.Fatalf("%+v", err)
		}

		if tks[0].ID != tks2[0].ID {
			tr.Fatalf("unexpected identity: %s (expected: %s)", tks[0].ID, tks2[0].ID)
		}
	})

	t.Run("Single", func(tr *testing.T) {
		tk := &model.Task{
			Desc:       fmt.Sprintf("%s%d", desc, 1001),
			Created:    now,
			Done:       true,
			Done2:      false,
			Count:      11,
			Count64:    11,
			Proportion: 0.12345 + 11,
			NameList:   []string{"a", "b", "c"},
			Flag: map[string]float64{
				"1": 1.1,
				"2": 2.2,
				"3": 3.3,
			},
		}
		id, err := taskRepo.Insert(ctx, tk)
		if err != nil {
			tr.Fatalf("%+v", err)
		}
		ids = append(ids, id)

		tk.Count = 12
		tk.Flag["4"] = 4.4
		if err := taskRepo.Update(ctx, tk); err != nil {
			tr.Fatalf("%+v", err)
		}

		tsk, err := taskRepo.Get(ctx, tk.ID)
		if err != nil {
			tr.Fatalf("%+v", err)
		}

		if tsk.Count != 12 {
			tr.Fatalf("unexpected Count: %d (expected: %d)", tsk.Count, 12)
		}

		if _, ok := tsk.Flag["4"]; !ok {
			tr.Fatalf("unexpected Flag: %v (expected: %v)", ok, true)
		}
	})
}

func TestFirestoreTransaction_Single(t *testing.T) {
	client := initFirestoreClient(t)

	taskRepo := model.NewTaskRepository(client)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	var ids []string
	defer func() {
		defer cancel()
		if err := taskRepo.DeleteMultiByIDs(ctx, ids); err != nil {
			t.Fatal(err)
		}
	}()

	now := time.Unix(0, time.Now().UnixNano())
	desc := "Hello, World!"

	t.Run("Insert", func(tr *testing.T) {
		err := client.RunTransaction(ctx, func(cx context.Context, tx *firestore.Transaction) error {
			tk := &model.Task{
				Desc:       fmt.Sprintf("%s01", desc),
				Created:    now,
				Done:       true,
				Done2:      false,
				Count:      10,
				Count64:    11,
				Proportion: 0.12345 + 11,
				NameList:   []string{"a", "b", "c"},
				Flag: map[string]float64{
					"1": 1.1,
					"2": 2.2,
					"3": 3.3,
				},
			}

			id, err := taskRepo.InsertWithTx(cx, tx, tk)
			if err != nil {
				return err
			}

			ids = append(ids, id)
			return nil
		})

		if err != nil {
			tr.Fatalf("error: %+v", err)
		}

		tsk, err := taskRepo.Get(ctx, ids[len(ids)-1])
		if err != nil {
			tr.Fatalf("%+v", err)
		}

		if tsk.Count != 10 {
			tr.Fatalf("unexpected Count: %d (expected: %d)", tsk.Count, 10)
		}
	})

	t.Run("Update", func(tr *testing.T) {
		tkID := ids[len(ids)-1]
		err := client.RunTransaction(ctx, func(cx context.Context, tx *firestore.Transaction) error {
			tk, err := taskRepo.GetWithTx(tx, tkID)
			if err != nil {
				return err
			}

			if tk.Count != 10 {
				return fmt.Errorf("unexpected Count: %d (expected: %d)", tk.Count, 10)
			}

			tk.Count = 11
			if err = taskRepo.UpdateWithTx(cx, tx, tk); err != nil {
				return err
			}
			return nil
		})

		if err != nil {
			tr.Fatalf("error: %+v", err)
		}

		tsk, err := taskRepo.Get(ctx, tkID)
		if err != nil {
			tr.Fatalf("%+v", err)
		}

		if tsk.Count != 11 {
			tr.Fatalf("unexpected Count: %d (expected: %d)", tsk.Count, 11)
		}
	})
}

func TestFirestoreQuery(t *testing.T) {
	client := initFirestoreClient(t)

	taskRepo := model.NewTaskRepository(client)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	var ids []string
	defer func() {
		defer cancel()
		if err := taskRepo.DeleteMultiByIDs(ctx, ids); err != nil {
			t.Fatalf("%+v\n", err)
		}
	}()

	now := time.Unix(0, time.Now().UnixNano())
	desc := "Hello, World!"

	tks := make([]*model.Task, 0)
	for i := 1; i <= 10; i++ {
		tk := &model.Task{
			ID:         fmt.Sprintf("%d", i),
			Desc:       fmt.Sprintf("%s%d", desc, i),
			Created:    now,
			Done:       true,
			Done2:      false,
			Count:      i,
			Count64:    int64(i),
			Proportion: 0.12345 + float64(i),
			NameList:   []string{"a", "b", "c"},
			Flag: map[string]float64{
				"1": 1.1,
				"2": 2.2,
				"3": 3.3,
				"4": 4.4,
				"5": 5.5,
			},
		}
		tks = append(tks, tk)
	}
	ids, err := taskRepo.InsertMulti(ctx, tks)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	t.Run("int(1件)", func(tr *testing.T) {
		req := &model.TaskListReq{
			Count: model.NewQueryChainer().Equal(1),
		}

		tasks, err := taskRepo.List(ctx, req, nil)
		if err != nil {
			tr.Fatalf("%+v", err)
		}

		if len(tasks) != 1 {
			tr.Fatalf("unexpected length: %d (expected: %d)", len(tasks), 10)
		}
	})

	t.Run("int64(5件)", func(tr *testing.T) {
		req := &model.TaskListReq{
			Count64: model.NewQueryChainer().LessThanOrEqual(5),
		}

		tasks, err := taskRepo.List(ctx, req, nil)
		if err != nil {
			tr.Fatalf("%+v", err)
		}

		if len(tasks) != 5 {
			tr.Fatalf("unexpected length: %d (expected: %d)", len(tasks), 5)
		}
	})

	t.Run("float(1件)", func(tr *testing.T) {
		req := &model.TaskListReq{
			Proportion: model.NewQueryChainer().Equal(1.12345),
		}

		tasks, err := taskRepo.List(ctx, req, nil)
		if err != nil {
			tr.Fatalf("%+v", err)
		}

		if len(tasks) != 1 {
			tr.Fatalf("unexpected length: %d (expected: %d)", len(tasks), 1)
		}
	})

	t.Run("bool(10件)", func(tr *testing.T) {
		req := &model.TaskListReq{
			Done: model.NewQueryChainer().Equal(true),
		}

		tasks, err := taskRepo.List(ctx, req, nil)
		if err != nil {
			tr.Fatalf("%+v", err)
		}

		if len(tasks) != 10 {
			tr.Fatalf("unexpected length: %d (expected: %d)", len(tasks), 10)
		}
	})

	t.Run("time.Time(10件)", func(tr *testing.T) {
		req := &model.TaskListReq{
			Created: model.NewQueryChainer().Equal(now),
		}

		tasks, err := taskRepo.List(ctx, req, nil)
		if err != nil {
			tr.Fatalf("%+v", err)
		}

		if len(tasks) != 10 {
			tr.Fatalf("unexpected length: %d (expected: %d)", len(tasks), 10)
		}
	})

	t.Run("[]string(10件)", func(tr *testing.T) {
		req := &model.TaskListReq{
			NameList: model.NewQueryChainer().ArrayContainsAny([]string{"a", "b"}),
		}

		tasks, err := taskRepo.List(ctx, req, nil)
		if err != nil {
			tr.Fatalf("%+v", err)
		}

		if len(tasks) != 10 {
			tr.Fatalf("unexpected length: %d (expected: %d)", len(tasks), 10)
		}
	})

	t.Run("Flag(10件)", func(tr *testing.T) {
		req := &model.TaskListReq{
			Flag: model.NewQueryChainer().Equal(map[string]float64{
				"1": 1.1,
				"2": 2.2,
				"3": 3.3,
			}),
		}

		tasks, err := taskRepo.List(ctx, req, nil)
		if err != nil {
			tr.Fatalf("%+v", err)
		}

		if len(tasks) != 10 {
			tr.Fatalf("unexpected length: %d (expected: %d)", len(tasks), 10)
		}
	})

	t.Run("UseQueryBuilder", func(tr *testing.T) {
		qb := model.NewQueryBuilder(taskRepo.GetCollection())
		qb.GreaterThan("count", 3)
		qb.LessThan("count", 8)

		tasks, err := taskRepo.List(ctx, nil, qb.Query())
		if err != nil {
			tr.Fatalf("%+v", err)
		}

		if len(tasks) != 4 {
			tr.Fatalf("unexpected length: %d (expected: %d)", len(tasks), 4)
		}
	})
}

func TestFirestoreError(t *testing.T) {
	client := initFirestoreClient(t)

	taskRepo := model.NewTaskRepository(client)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	var ids []string
	defer func() {
		defer cancel()
		if err := taskRepo.DeleteMultiByIDs(ctx, ids); err != nil {
			t.Fatalf("%+v\n", err)
		}
	}()

	now := time.Unix(0, time.Now().UnixNano())
	desc := "Hello, World!"

	t.Run("Prepare", func(tr *testing.T) {
		tk := &model.Task{
			Desc:       desc,
			Created:    now,
			Done:       true,
			Done2:      false,
			Count:      11,
			Count64:    11,
			Proportion: 0.12345 + 11,
			NameList:   []string{"a", "b", "c"},
			Flag: map[string]float64{
				"1": 1.1,
				"2": 2.2,
				"3": 3.3,
			},
		}
		id, err := taskRepo.Insert(ctx, tk)
		if err != nil {
			tr.Fatalf("%+v", err)
		}
		ids = append(ids, id)
	})

	t.Run("ErrorReadAfterWrite", func(tr *testing.T) {
		tkID := ids[len(ids)-1]
		errReadAfterWrite := xerrors.New("firestore: read after write in transaction")

		if err := client.RunTransaction(ctx, func(cx context.Context, tx *firestore.Transaction) error {
			tk, err := taskRepo.GetWithTx(tx, tkID)
			if err != nil {
				return err
			}

			if tk.Count != 11 {
				return fmt.Errorf("unexpected Count: %d (expected: %d)", tk.Count, 11)
			}

			tk.Count = 12
			if err = taskRepo.UpdateWithTx(cx, tx, tk); err != nil {
				return err
			}

			if _, err = taskRepo.GetWithTx(tx, tkID); err != nil {
				return err
			}
			return nil
		}); err != nil && xerrors.Is(xerrors.Unwrap(err), errReadAfterWrite) {
			tr.Fatalf("error: %+v", err)
		}

		tsk, err := taskRepo.Get(ctx, tkID)
		if err != nil {
			tr.Fatalf("%+v", err)
		}

		if tsk.Count != 11 {
			tr.Fatalf("unexpected Count: %d (expected: %d)", tsk.Count, 11)
		}

		if err = client.RunTransaction(ctx, func(cx context.Context, tx *firestore.Transaction) error {
			id, err := taskRepo.InsertWithTx(cx, tx, new(model.Task))
			if err != nil {
				return err
			}

			if _, err = taskRepo.GetWithTx(tx, id); err != nil {
				return err
			}
			return nil
		}); err != nil && xerrors.Is(xerrors.Unwrap(err), errReadAfterWrite) {
			tr.Fatalf("error: %+v", err)
		}
	})
}

/* TODO Map版Indexes実装
func TestFirestoreListNameWithIndexes(t *testing.T) {
	client := initFirestoreClient(t)

	nameRepo := model.NewNameRepository(client)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	var ids []string
	defer func() {
		defer cancel()
		if err := nameRepo.DeleteMultiByIDs(ctx, ids); err != nil {
			t.Fatal(err)
		}
	}()

	now := time.Unix(0, time.Now().UnixNano())
	desc := "Hello, World!"
	desc2 := "Prefix, Test!"

	tks := make([]*model.Name, 0)
	for i := int64(1); i <= 10; i++ {
		tk := &model.Name{
			ID:        i,
			Created:   now,
			Desc:      fmt.Sprintf("%s%d", desc, i),
			Desc2:     fmt.Sprintf("%s%d", desc2, i),
			Done:      true,
			Count:     int(i),
			PriceList: []int{1, 2, 3, 4, 5},
		}
		tks = append(tks, tk)
	}
	ids, err := nameRepo.InsertMulti(ctx, tks)
	if err != nil {
		tr.Fatalf("%+v", err)
	}

	t.Run("int(1件)", func(tr *testing.T) {
		req := &model.NameListReq{
			Count: model.NumericCriteriaBase.Parse(1),
		}

		tasks, err := nameRepo.List(ctx, req, nil)
		if err != nil {
			tr.Fatalf("%+v", err)
		}

		if len(tasks) != 1 {
			tr.Fatalf("unexpected length: %d (expected: %d)", len(tasks), 1)
		}
	})

	t.Run("bool(10件)", func(tr *testing.T) {
		req := &model.NameListReq{
			Done: model.BoolCriteriaTrue,
		}

		tasks, err := nameRepo.List(ctx, req, nil)
		if err != nil {
			tr.Fatalf("%+v", err)
		}

		if len(tasks) != 10 {
			tr.Fatalf("unexpected length: %d (expected: %d)", len(tasks), 10)
		}
	})

	t.Run("like(10件)", func(tr *testing.T) {
		req := &model.NameListReq{
			Desc: "ll",
		}

		tasks, err := nameRepo.List(ctx, req, nil)
		if err != nil {
			tr.Fatalf("%+v", err)
		}

		if len(tasks) != 10 {
			tr.Fatalf("unexpected length: %d (expected: %d)", len(tasks), 10)
		}
	})

	t.Run("prefix", func(tr *testing.T) {
		t.Run("Success", func(tr *testing.T) {
			req := &model.NameListReq{
				Desc2: "Pre",
			}

			tasks, err := nameRepo.List(ctx, req, nil)
			if err != nil {
				tr.Fatalf("%+v", err)
			}

			if len(tasks) != 10 {
				tr.Fatalf("unexpected length: %d (expected: %d)", len(tasks), 10)
			}
		})

		t.Run("Failure", func(tr *testing.T) {
			req := &model.NameListReq{
				Desc2: "He",
			}

			tasks, err := nameRepo.List(ctx, req, nil)
			if err != nil {
				tr.Fatalf("%+v", err)
			}

			if len(tasks) != 0 {
				tr.Fatalf("unexpected length: %d (expected: %d)", len(tasks), 0)
			}
		})
	})

	t.Run("time.Time(10件)", func(tr *testing.T) {
		req := &model.NameListReq{
			Created: now,
		}

		tasks, err := nameRepo.List(ctx, req, nil)
		if err != nil {
			tr.Fatalf("%+v", err)
		}

		if len(tasks) != 10 {
			tr.Fatalf("unexpected length: %d (expected: %d)", len(tasks), 10)
		}
	})

	t.Run("[]int(10件)", func(tr *testing.T) {
		req := &model.NameListReq{
			PriceList: []int{1, 2, 3},
		}

		tasks, err := nameRepo.List(ctx, req, nil)
		if err != nil {
			tr.Fatalf("%+v", err)
		}

		if len(tasks) != 10 {
			tr.Fatalf("unexpected length: %d (expected: %d)", len(tasks), 10)
		}
	})
}
*/

func TestFirestoreOfTaskRepo(t *testing.T) {
	client := initFirestoreClient(t)

	taskRepo := model.NewTaskRepository(client)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Unix(time.Now().Unix(), 0)
	desc := "hello"

	id, err := taskRepo.Insert(ctx, &model.Task{
		Desc:    desc,
		Created: now,
		Done:    true,
	})

	if err != nil {
		t.Fatalf("failed to put item: %+v", err)
	}

	ret, err := taskRepo.Get(ctx, id)

	if err != nil {
		t.Fatalf("failed to get item: %+v", err)
	}

	compareTask(t, &model.Task{
		ID:      id,
		Desc:    desc,
		Created: now,
		Done:    true,
	}, ret)

	returns, err := taskRepo.GetMulti(ctx, []string{id})

	if err != nil {
		t.Fatalf("failed to get item: %+v", err)
	}

	if len(returns) != 1 {
		t.Fatalf("GetMulti should return 1 item: %#v", returns)
	}

	compareTask(t, &model.Task{
		ID:      id,
		Desc:    desc,
		Created: now,
		Done:    true,
	}, returns[0])

	compareTask(t, &model.Task{
		ID:      id,
		Desc:    desc,
		Created: now,
		Done:    true,
	}, ret)

	if err := taskRepo.DeleteByID(ctx, id); err != nil {
		t.Fatalf("delete failed: %+v", err)
	}

	if _, err := taskRepo.Get(ctx, id); err == nil {
		t.Fatalf("should get an error")
	}
}

func TestFirestoreOfLockRepo(t *testing.T) {
	client := initFirestoreClient(t)

	lockRepo := model.NewLockRepository(client)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	ids := make([]string, 0)
	defer func() {
		defer cancel()
		if err := lockRepo.DeleteMultiByIDs(ctx, ids); err != nil {
			t.Fatal(err)
		}
	}()

	text := "hello"

	t.Run("insert_test", func(tr *testing.T) {
		l := &model.Lock{
			Text: text,
			Flag: nil,
			Meta: model.Meta{},
		}

		id, err := lockRepo.Insert(ctx, l)
		if err != nil {
			tr.Fatalf("failed to put item: %+v", err)
		}

		ids = append(ids, id)

		ret, err := lockRepo.Get(ctx, id)

		if err != nil {
			tr.Fatalf("failed to get item: %+v", err)
		}

		if text != ret.Text {
			tr.Fatalf("unexpected text: %s (expected: %s)", text, ret.Text)
		}
		if ret.CreatedAt.IsZero() {
			tr.Fatalf("unexpected createdAt zero:")
		}
		if ret.UpdatedAt.IsZero() {
			tr.Fatalf("unexpected updatedAt zero:")
		}
	})

	t.Run("update_test", func(tr *testing.T) {
		l := &model.Lock{
			Text: text,
			Flag: nil,
			Meta: model.Meta{},
		}

		id, err := lockRepo.Insert(ctx, l)
		if err != nil {
			tr.Fatalf("failed to put item: %+v", err)
		}

		ids = append(ids, id)

		time.Sleep(1 * time.Second)

		text = "hello!!!"
		l.Text = text
		err = lockRepo.Update(ctx, l)
		if err != nil {
			tr.Fatalf("failed to update item: %+v", err)
		}

		ret, err := lockRepo.Get(ctx, id)
		if err != nil {
			tr.Fatalf("failed to get item: %+v", err)
		}

		if text != ret.Text {
			tr.Fatalf("unexpected text: %s (expected: %s)", text, ret.Text)
		}
		if ret.CreatedAt.Unix() == ret.UpdatedAt.Unix() {
			tr.Fatalf("unexpected createdAt == updatedAt: %d == %d",
				ret.CreatedAt.Unix(), ret.UpdatedAt.Unix())
		}
	})

	t.Run("soft_delete_test", func(tr *testing.T) {
		l := &model.Lock{
			Text: text,
			Flag: nil,
			Meta: model.Meta{},
		}

		id, err := lockRepo.Insert(ctx, l)
		if err != nil {
			tr.Fatalf("failed to put item: %+v", err)
		}

		ids = append(ids, id)

		l.Text = text
		err = lockRepo.Delete(ctx, l, model.DeleteOption{
			Mode: model.DeleteModeSoft,
		})
		if err != nil {
			tr.Fatalf("failed to soft delete item: %+v", err)
		}

		ret, err := lockRepo.Get(ctx, id, model.GetOption{
			IncludeSoftDeleted: true,
		})
		if err != nil {
			tr.Fatalf("failed to get item: %+v", err)
		}

		if text != ret.Text {
			tr.Fatalf("unexpected text: %s (expected: %s)", text, ret.Text)
		}
		if ret.DeletedAt == nil {
			tr.Fatalf("unexpected deletedAt == nil: %+v", ret.DeletedAt)
		}
	})

	t.Run("hard_delete_test", func(tr *testing.T) {
		l := &model.Lock{
			Text: text,
			Flag: nil,
			Meta: model.Meta{},
		}

		id, err := lockRepo.Insert(ctx, l)
		if err != nil {
			tr.Fatalf("failed to put item: %+v", err)
		}

		l.Text = text
		err = lockRepo.Delete(ctx, l)
		if err != nil {
			tr.Fatalf("failed to hard delete item: %+v", err)
		}

		ret, err := lockRepo.Get(ctx, id, model.GetOption{
			IncludeSoftDeleted: true,
		})
		if err != nil && !strings.Contains(err.Error(), "not found") {
			tr.Fatalf("failed to get item: %+v", err)
		}

		if ret != nil {
			tr.Fatalf("failed to delete item (found!): %+v", ret)
		}
	})

	t.Run("UseQueryBuilder", func(tr *testing.T) {
		l := &model.Lock{
			Text: text,
			Flag: nil,
			Meta: model.Meta{},
		}
		id, err := lockRepo.Insert(ctx, l)
		if err != nil {
			tr.Fatalf("failed to put item: %+v", err)
		}

		ids = append(ids, id)

		qb := model.NewQueryBuilder(lockRepo.GetCollection())
		qb.GreaterThanOrEqual("CreatedAt", model.SetLastThreeToZero(l.CreatedAt).Add(-100))
		qb.LessThanOrEqual("CreatedAt", model.SetLastThreeToZero(l.CreatedAt).Add(100))
		if err = qb.Check(); err != nil {
			tr.Fatal(err)
		}

		locks, err := lockRepo.List(ctx, nil, qb.Query())
		if err != nil {
			tr.Fatalf("%+v", err)
		}

		if len(locks) != 1 {
			tr.Fatalf("unexpected length: %d (expected: %d)", len(locks), 1)
		}

		if id != locks[0].ID {
			tr.Fatalf("unexpected length: %s (expected: %s)", locks[0].ID, id)
		}
	})

	t.Run("UseQueryChainer", func(tr *testing.T) {
		l := &model.Lock{
			Text: "Hello",
			Flag: nil,
			Meta: model.Meta{},
		}
		id, err := lockRepo.Insert(ctx, l)
		if err != nil {
			tr.Fatalf("failed to put item: %+v", err)
		}
		ids = append(ids, id)
		l = &model.Lock{
			Text: "World",
			Flag: nil,
			Meta: model.Meta{},
		}
		id, err = lockRepo.Insert(ctx, l)
		if err != nil {
			tr.Fatalf("failed to put item: %+v", err)
		}
		ids = append(ids, id)
		req := &model.LockListReq{
			Text:               model.NewQueryChainer().In([]string{"Hello", "World"}),
			IncludeSoftDeleted: true,
		}
		locks, err := lockRepo.List(ctx, req, nil)
		if err != nil {
			tr.Fatalf("%+v", err)
		}
		if len(locks) != 2 {
			tr.Fatalf("unexpected length: %d (expected: %d)", len(locks), 2)
		}

		now := time.Now()
		req = &model.LockListReq{
			CreatedAt:          model.NewQueryChainer().GreaterThanOrEqual(now.Add(time.Second * 5 * -1)).LessThanOrEqual(now.Add(time.Second * 5)),
			IncludeSoftDeleted: true,
		}
		locks, err = lockRepo.List(ctx, req, nil)
		if err != nil {
			tr.Fatalf("%+v", err)
		}
		if len(locks) != 6 {
			tr.Fatalf("unexpected length: %d (expected: %d)", len(locks), 6)
		}
	})
}
