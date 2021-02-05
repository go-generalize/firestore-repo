// +build internal

package tests

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	model "github.com/go-generalize/firestore-repo/testfiles/not_auto"
	"golang.org/x/xerrors"
	"google.golang.org/genproto/googleapis/type/latlng"
)

func initFirestoreClient(t *testing.T) *firestore.Client {
	t.Helper()

	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		os.Setenv("FIRESTORE_EMULATOR_HOST", "localhost:8000")
	}

	os.Setenv("FIRESTORE_PROJECT_ID", "project-id-in-google")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := firestore.NewClient(ctx, "testing2")

	if err != nil {
		t.Fatalf("failed to initialize firestore client: %+v", err)
	}

	return client
}

func compareTask(t *testing.T, expected, actual *model.Task) {
	t.Helper()

	if actual.Identity != expected.Identity {
		t.Fatalf("unexpected identity: %s (expected: %s)", actual.Identity, expected.Identity)
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
		if err := taskRepo.DeleteMultiByIdentities(ctx, ids); err != nil {
			t.Fatal(err)
		}
	}()

	now := time.Unix(0, time.Now().UnixNano()).UTC()
	desc := "Hello, World!"

	t.Run("Multi", func(tr *testing.T) {
		tks := make([]*model.Task, 0)
		for i := int64(1); i <= 10; i++ {
			tk := &model.Task{
				Identity:   fmt.Sprintf("Task_%d", i),
				Desc:       fmt.Sprintf("%s%d", desc, i),
				Created:    now,
				Done:       true,
				Done2:      false,
				Count:      int(i),
				Count64:    0,
				Proportion: 0.12345 + float64(i),
				NameList:   []string{"a", "b", "c"},
				Flag:       model.Flag(true),
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
				Identity:   ids[i-1],
				Desc:       fmt.Sprintf("%s%d", desc, i),
				Created:    now,
				Done:       true,
				Done2:      false,
				Count:      int(i),
				Count64:    i,
				Proportion: 0.12345 + float64(i),
				NameList:   []string{"a", "b", "c"},
				Flag:       model.Flag(true),
			}
			tks2 = append(tks2, tk)
		}
		if err := taskRepo.UpdateMulti(ctx, tks2); err != nil {
			tr.Fatalf("%+v", err)
		}

		if tks[0].Identity != tks2[0].Identity {
			tr.Fatalf("unexpected identity: %s (expected: %s)", tks[0].Identity, tks2[0].Identity)
		}
	})

	t.Run("Single", func(tr *testing.T) {
		tk := &model.Task{
			Identity:   "Single",
			Desc:       fmt.Sprintf("%s%d", desc, 1001),
			Created:    now,
			Done:       true,
			Done2:      false,
			Count:      11,
			Count64:    11,
			Proportion: 0.12345 + 11,
			NameList:   []string{"a", "b", "c"},
			Flag:       model.Flag(true),
		}
		id, err := taskRepo.Insert(ctx, tk)
		if err != nil {
			tr.Fatalf("%+v", err)
		}
		ids = append(ids, id)

		tr.Run("SubCollection", func(tr *testing.T) {
			ids2 := make([]string, 0, 3)
			doc := taskRepo.GetDocRef(id)
			subRepo := model.NewSubTaskRepository(client, doc)
			st := &model.SubTask{IsSubCollection: true}
			id, err = subRepo.Insert(ctx, st)
			if err != nil {
				tr.Fatalf("%+v", err)
			}
			ids2 = append(ids2, id)

			sts := []*model.SubTask{
				{IsSubCollection: true},
				{IsSubCollection: false},
			}
			stsIDs, err := subRepo.InsertMulti(ctx, sts)
			if err != nil {
				tr.Fatalf("%+v", err)
			}
			ids2 = append(ids2, stsIDs...)

			listReq := &model.SubTaskListReq{IsSubCollection: model.NewQueryChainer().Equal(true)}
			sts, err = subRepo.List(ctx, listReq, nil)
			if err != nil {
				tr.Fatalf("%+v", err)
			}

			if len(sts) != 2 {
				tr.Fatal("not match")
			}

			tr.Run("Reference", func(tr2 *testing.T) {
				tk.Sub = subRepo.GetDocRef(sts[1].ID)
				if err := taskRepo.Update(ctx, tk); err != nil {
					tr2.Fatalf("%+v", err)
				}

				tkr, err := taskRepo.Get(ctx, doc.ID)
				if err != nil {
					tr2.Fatalf("%+v", err)
				}

				sub, err := subRepo.GetWithDoc(ctx, tkr.Sub)
				if err != nil {
					tr2.Fatalf("%+v", err)
				}

				if sub.ID != sts[1].ID {
					tr2.Fatal("not match")
				}

				taskListReq := &model.TaskListReq{Sub: model.NewQueryChainer().Equal(tk.Sub)}
				tks, err := taskRepo.List(ctx, taskListReq, nil)
				if err != nil {
					tr2.Fatalf("%+v", err)
				}
				if len(tks) != 1 {
					tr2.Fatal("not match")
				}
			})

			if err = subRepo.DeleteMultiByIDs(ctx, ids2); err != nil {
				tr.Fatalf("%+v", err)
			}
		})

		tk.Count = 12
		if err := taskRepo.Update(ctx, tk); err != nil {
			tr.Fatalf("%+v", err)
		}

		tsk, err := taskRepo.Get(ctx, tk.Identity)
		if err != nil {
			tr.Fatalf("%+v", err)
		}

		if tsk.Count != 12 {
			tr.Fatalf("unexpected Count: %d (expected: %d)", tsk.Count, 12)
		}

		tr.Run("UpdateBuilder", func(ttr *testing.T) {
			desc1002 := fmt.Sprintf("%s%d", desc, 1002)

			updateParam := &model.TaskUpdateParam{
				Desc:       model.NewUpdater(desc1002),
				Created:    model.NewUpdater(firestore.ServerTimestamp),
				Done:       model.NewUpdater(false),
				Count:      model.NewUpdater(firestore.Increment(1)),
				Count64:    model.NewUpdater(firestore.Increment(2)),
				Proportion: model.NewUpdater(firestore.Increment(0.1)),
			}

			if err = taskRepo.StrictUpdate(ctx, tsk.Identity, updateParam); err != nil {
				ttr.Fatalf("%+v", err)
			}

			tsk, err = taskRepo.Get(ctx, tk.Identity)
			if err != nil {
				ttr.Fatalf("%+v", err)
			}

			if tsk.Desc != desc1002 {
				ttr.Fatalf("unexpected Desc: %s (expected: %s)", tsk.Desc, desc1002)
			}

			if tsk.Created.Before(now) {
				ttr.Fatalf("unexpected Created > now: %t (expected: %t)", tsk.Created.Before(now), tsk.Created.After(now))
			}

			if tsk.Done {
				ttr.Fatalf("unexpected Done: %t (expected: %t)", tsk.Done, false)
			}

			if tsk.Count != 13 {
				ttr.Fatalf("unexpected Count: %d (expected: %d)", tsk.Count, 13)
			}

			if tsk.Count64 != 13 {
				ttr.Fatalf("unexpected Count64: %d (expected: %d)", tsk.Count64, 13)
			}

			if tsk.Proportion != 11.22345 {
				ttr.Fatalf("unexpected Proportion: %g (expected: %g)", tsk.Proportion, 11.22345)
			}
		})
	})
}

func TestFirestoreTransaction_Single(t *testing.T) {
	client := initFirestoreClient(t)

	taskRepo := model.NewTaskRepository(client)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	var ids []string
	defer func() {
		defer cancel()
		if err := taskRepo.DeleteMultiByIdentities(ctx, ids); err != nil {
			t.Fatal(err)
		}
	}()

	now := time.Unix(0, time.Now().UnixNano())
	desc := "Hello, World!"
	latLng := &latlng.LatLng{
		Latitude:  35.678803,
		Longitude: 139.756263,
	}

	t.Run("Insert", func(tr *testing.T) {
		if err := client.RunTransaction(ctx, func(cx context.Context, tx *firestore.Transaction) error {
			tk := &model.Task{
				Identity:   "identity",
				Desc:       fmt.Sprintf("%s01", desc),
				Created:    now,
				Done:       true,
				Done2:      false,
				Count:      10,
				Count64:    11,
				NameList:   []string{"a", "b", "c"},
				Proportion: 0.12345 + 11,
				Geo:        latLng,
				Flag:       true,
			}

			id, err := taskRepo.InsertWithTx(cx, tx, tk)
			if err != nil {
				return err
			}

			ids = append(ids, id)
			return nil
		}); err != nil {
			tr.Fatalf("error: %+v", err)
		}

		tsk, err := taskRepo.Get(ctx, ids[len(ids)-1])
		if err != nil {
			tr.Fatalf("%+v", err)
		}

		if reflect.DeepEqual(tsk.Geo, latLng) {
			tr.Fatalf("unexpected Geo: %+v (expected: %+v)", tsk.Geo, latLng)
		}
	})

	t.Run("Update", func(tr *testing.T) {
		id := ids[len(ids)-1]
		if err := client.RunTransaction(ctx, func(cx context.Context, tx *firestore.Transaction) error {
			tk, err := taskRepo.GetWithTx(tx, id)
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
		}); err != nil {
			tr.Fatalf("error: %+v", err)
		}

		tsk, err := taskRepo.Get(ctx, id)
		if err != nil {
			tr.Fatalf("%+v", err)
		}

		if tsk.Count != 11 {
			tr.Fatalf("unexpected Count: %d (expected: %d)", tsk.Count, 11)
		}
	})
}

func TestFirestoreTransaction_Multi(t *testing.T) {
	client := initFirestoreClient(t)

	taskRepo := model.NewTaskRepository(client)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	var ids []string
	defer func() {
		defer cancel()
		if err := taskRepo.DeleteMultiByIdentities(ctx, ids); err != nil {
			t.Fatal(err)
		}
	}()

	now := time.Unix(0, time.Now().UnixNano())
	desc := "Hello, World!"
	latLng := &latlng.LatLng{
		Latitude:  35.678803,
		Longitude: 139.756263,
	}

	tks := make([]*model.Task, 0)
	t.Run("InsertMulti", func(tr *testing.T) {
		for i := int64(1); i <= 10; i++ {
			tk := &model.Task{
				Identity:   fmt.Sprintf("Task_%d", i),
				Desc:       fmt.Sprintf("%s%d", desc, i),
				Created:    now,
				Done:       true,
				Done2:      false,
				Count:      int(i),
				Count64:    0,
				NameList:   []string{"a", "b", "c"},
				Proportion: 0.12345 + float64(i),
				Geo:        latLng,
				Flag:       model.Flag(true),
			}
			tks = append(tks, tk)
		}

		if err := client.RunTransaction(ctx, func(cx context.Context, tx *firestore.Transaction) error {
			idList, err := taskRepo.InsertMultiWithTx(ctx, tx, tks)
			if err != nil {
				return err
			}
			ids = append(ids, idList...)
			return nil
		}); err != nil {
			tr.Fatalf("error: %+v", err)
		}
	})

	tks2 := make([]*model.Task, 0)
	t.Run("UpdateMulti", func(tr *testing.T) {
		for i := int64(1); i <= 10; i++ {
			tk := &model.Task{
				Identity:   ids[i-1],
				Desc:       fmt.Sprintf("%s%d", desc, i+1),
				Created:    now,
				Done:       false,
				Done2:      true,
				Count:      int(i),
				Count64:    i,
				NameList:   []string{"a", "b", "c"},
				Proportion: 0.12345 + float64(i),
				Geo:        latLng,
				Flag:       model.Flag(true),
			}
			tks2 = append(tks2, tk)
		}

		if err := client.RunTransaction(ctx, func(cx context.Context, tx *firestore.Transaction) error {
			if err := taskRepo.UpdateMultiWithTx(cx, tx, tks2); err != nil {
				return err
			}
			return nil
		}); err != nil {
			tr.Fatalf("error: %+v", err)
		}

		if tks[0].Identity != tks2[0].Identity {
			tr.Fatalf("unexpected identity: %s (expected: %s)", tks[0].Identity, tks2[0].Identity)
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
		if err := taskRepo.DeleteMultiByIdentities(ctx, ids); err != nil {
			t.Fatalf("%+v\n", err)
		}
	}()

	now := time.Unix(0, time.Now().UnixNano())
	desc := "Hello, World!"
	latLng := &latlng.LatLng{
		Latitude:  35.678803,
		Longitude: 139.756263,
	}
	skyTreeLatLng := &latlng.LatLng{
		Latitude:  35.7100069,
		Longitude: 139.8108103,
	}

	tks := make([]*model.Task, 0)
	for i := 1; i <= 9; i++ {
		tk := &model.Task{
			Identity:   fmt.Sprintf("%d", i),
			Desc:       fmt.Sprintf("%s%d", desc, i),
			Created:    now,
			Done:       true,
			Done2:      false,
			Count:      i,
			Count64:    int64(i),
			NameList:   []string{"a", "b", "c"},
			Proportion: 0.12345 + float64(i),
			Geo:        latLng,
			Flag:       model.Flag(true),
		}
		tks = append(tks, tk)
	}

	{
		tk := &model.Task{
			Identity:   fmt.Sprintf("%d", 10),
			Desc:       fmt.Sprintf("%s%d", desc, 10),
			Created:    now,
			Done:       true,
			Done2:      false,
			Count:      10,
			Count64:    10,
			NameList:   []string{"a", "b", "c"},
			Proportion: 10.12345,
			Geo:        skyTreeLatLng,
			Flag:       model.Flag(true),
		}
		tks = append(tks, tk)
	}

	ids, err := taskRepo.InsertMulti(ctx, tks)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	t.Run("int(1件)", func(t *testing.T) {
		req := &model.TaskListReq{
			Count: model.NewQueryChainer().Equal(1),
		}

		tasks, err := taskRepo.List(ctx, req, nil)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		if len(tasks) != 1 {
			t.Fatal("not match")
		}
	})

	t.Run("int64(6件)", func(tr *testing.T) {
		req := &model.TaskListReq{
			Count64: model.NewQueryChainer().GreaterThanOrEqual(5),
		}

		tasks, err := taskRepo.List(ctx, req, nil)
		if err != nil {
			tr.Fatalf("%+v", err)
		}

		if len(tasks) != 6 {
			tr.Fatalf("unexpected length: %d (expected: %d)", len(tasks), 6)
		}
	})

	t.Run("float(1件)", func(t *testing.T) {
		req := &model.TaskListReq{
			Proportion: model.NewQueryChainer().Equal(1.12345),
		}

		tasks, err := taskRepo.List(ctx, req, nil)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		if len(tasks) != 1 {
			t.Fatal("not match")
		}
	})

	t.Run("bool(10件)", func(t *testing.T) {
		req := &model.TaskListReq{
			Done: model.NewQueryChainer().Equal(true),
		}

		tasks, err := taskRepo.List(ctx, req, nil)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		if len(tasks) != 10 {
			t.Fatal("not match")
		}
	})

	t.Run("time.Time(10件)", func(t *testing.T) {
		req := &model.TaskListReq{
			Created: model.NewQueryChainer().Equal(now),
		}

		tasks, err := taskRepo.List(ctx, req, nil)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		if len(tasks) != 10 {
			t.Fatal("not match")
		}
	})

	t.Run("[]string(10件)", func(t *testing.T) {
		req := &model.TaskListReq{
			NameList: model.NewQueryChainer().ArrayContainsAny([]string{"a", "b"}),
		}

		tasks, err := taskRepo.List(ctx, req, nil)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		if len(tasks) != 10 {
			t.Fatal("not match")
		}
	})

	t.Run("Flag(10件)", func(t *testing.T) {
		req := &model.TaskListReq{
			Flag: model.NewQueryChainer().Equal(true),
		}

		tasks, err := taskRepo.List(ctx, req, nil)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		if len(tasks) != 10 {
			t.Fatal("not match")
		}
	})

	t.Run("Geo(9件)", func(t *testing.T) {
		req := &model.TaskListReq{
			Geo: model.NewQueryChainer().Equal(latLng),
		}

		tasks, err := taskRepo.List(ctx, req, nil)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		if len(tasks) != 9 {
			t.Fatal("not match")
		}
	})

	t.Run("NotEqual(1件)", func(tr *testing.T) {
		req := &model.TaskListReq{
			Geo: model.NewQueryChainer().NotEqual(latLng),
		}
		tasks, err := taskRepo.List(ctx, req, nil)
		if err != nil {
			tr.Fatalf("%+v", err)
		}
		if len(tasks) != 1 {
			tr.Fatalf("unexpected length: %d (expected: %d)", len(tasks), 1)
		}
	})

	t.Run("NotIn(9件)", func(tr *testing.T) {
		req := &model.TaskListReq{
			Geo: model.NewQueryChainer().NotIn([]*latlng.LatLng{skyTreeLatLng}),
		}
		tasks, err := taskRepo.List(ctx, req, nil)
		if err != nil {
			tr.Fatalf("%+v", err)
		}
		if len(tasks) != 9 {
			tr.Fatalf("unexpected length: %d (expected: %d)", len(tasks), 9)
		}
	})

	t.Run("UseQueryBuilder", func(tr *testing.T) {
		tr.Run("range query(3<count<8)", func(ttr *testing.T) {
			qb := model.NewQueryBuilder(taskRepo.GetCollection())
			qb.GreaterThan("count", 3)
			qb.LessThan("count", 8)

			tasks, err := taskRepo.List(ctx, nil, qb.Query())
			if err != nil {
				ttr.Fatalf("%+v", err)
			}

			if len(tasks) != 4 {
				ttr.Fatalf("unexpected length: %d (expected: %d)", len(tasks), 4)
			}
		})
		tr.Run("!=(count!=1)", func(ttr *testing.T) {
			qb := model.NewQueryBuilder(taskRepo.GetCollection())
			qb.NotEqual("count", 1)

			tasks, err := taskRepo.List(ctx, nil, qb.Query())
			if err != nil {
				ttr.Fatalf("%+v", err)
			}

			if len(tasks) != 9 {
				ttr.Fatalf("unexpected length: %d (expected: %d)", len(tasks), 9)
			}
		})
		tr.Run("not-in(count not-in [1,2,3,4,5])", func(ttr *testing.T) {
			qb := model.NewQueryBuilder(taskRepo.GetCollection())
			qb.NotIn("count", []int{1, 2, 3, 4, 5})

			tasks, err := taskRepo.List(ctx, nil, qb.Query())
			if err != nil {
				ttr.Fatalf("%+v", err)
			}

			if len(tasks) != 5 {
				ttr.Fatalf("unexpected length: %d (expected: %d)", len(tasks), 5)
			}
		})
	})
}

func TestFirestoreError(t *testing.T) {
	client := initFirestoreClient(t)

	taskRepo := model.NewTaskRepository(client)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	var ids []string
	defer func() {
		defer cancel()
		if err := taskRepo.DeleteMultiByIdentities(ctx, ids); err != nil {
			t.Fatalf("%+v\n", err)
		}
	}()

	now := time.Unix(0, time.Now().UnixNano())
	desc := "Hello, World!"

	t.Run("Prepare", func(tr *testing.T) {
		tk := &model.Task{
			Identity:   "identity",
			Desc:       desc,
			Created:    now,
			Done:       true,
			Done2:      false,
			Count:      11,
			Count64:    11,
			Proportion: 0.12345 + 11,
			NameList:   []string{"a", "b", "c"},
			Flag:       model.Flag(true),
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
func TestFirestoreListWithIndexes(t *testing.T) {
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
			Identity:        i,
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
		t.Fatalf("%+v", err)
	}

	t.Run("int(1件)", func(t *testing.T) {
		req := &model.NameListReq{
			Count: model.NumericCriteriaBase.Parse(1),
		}

		tasks, err := nameRepo.List(ctx, req, nil)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		if len(tasks) != 1 {
			t.Fatal("not match")
		}
	})

	t.Run("bool(10件)", func(t *testing.T) {
		req := &model.NameListReq{
			Done: model.BoolCriteriaTrue,
		}

		tasks, err := nameRepo.List(ctx, req, nil)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		if len(tasks) != 10 {
			t.Fatal("not match")
		}
	})

	t.Run("like(10件)", func(t *testing.T) {
		req := &model.NameListReq{
			Desc: "ll",
		}

		tasks, err := nameRepo.List(ctx, req, nil)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		if len(tasks) != 10 {
			t.Fatal("not match")
		}
	})

	t.Run("prefix", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			req := &model.NameListReq{
				Desc2: "Pre",
			}

			tasks, err := nameRepo.List(ctx, req, nil)
			if err != nil {
				t.Fatalf("%+v", err)
			}

			if len(tasks) != 10 {
				t.Fatal("not match")
			}
		})

		t.Run("Failure", func(t *testing.T) {
			req := &model.NameListReq{
				Desc2: "He",
			}

			tasks, err := nameRepo.List(ctx, req, nil)
			if err != nil {
				t.Fatalf("%+v", err)
			}

			if len(tasks) != 0 {
				t.Fatal("not match")
			}
		})
	})

	t.Run("time.Time(10件)", func(t *testing.T) {
		req := &model.NameListReq{
			Created: now,
		}

		tasks, err := nameRepo.List(ctx, req, nil)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		if len(tasks) != 10 {
			t.Fatal("not match")
		}
	})

	t.Run("[]int(10件)", func(t *testing.T) {
		req := &model.NameListReq{
			PriceList: []int{1, 2, 3},
		}

		tasks, err := nameRepo.List(ctx, req, nil)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		if len(tasks) != 10 {
			t.Fatal("not match")
		}
	})
}
*/

func TestFirestoreValueCheck(t *testing.T) {
	client := initFirestoreClient(t)

	taskRepo := model.NewTaskRepository(client)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Unix(time.Now().Unix(), 0)
	desc := "hello"

	id, err := taskRepo.Insert(ctx, &model.Task{
		Identity: "TestID",
		Desc:     desc,
		Created:  now,
		Done:     true,
	})

	if err != nil {
		t.Fatalf("failed to put item: %+v", err)
	}

	ret, err := taskRepo.Get(ctx, id)

	if err != nil {
		t.Fatalf("failed to get item: %+v", err)
	}

	compareTask(t, &model.Task{
		Identity: id,
		Desc:     desc,
		Created:  now,
		Done:     true,
	}, ret)

	returns, err := taskRepo.GetMulti(ctx, []string{id})

	if err != nil {
		t.Fatalf("failed to get item: %+v", err)
	}

	if len(returns) != 1 {
		t.Fatalf("GetMulti should return 1 item: %#v", returns)
	}

	compareTask(t, &model.Task{
		Identity: id,
		Desc:     desc,
		Created:  now,
		Done:     true,
	}, returns[0])

	compareTask(t, &model.Task{
		Identity: id,
		Desc:     desc,
		Created:  now,
		Done:     true,
	}, ret)

	if err := taskRepo.DeleteByIdentity(ctx, id); err != nil {
		t.Fatalf("delete failed: %+v", err)
	}

	if _, err := taskRepo.Get(ctx, id); err == nil {
		t.Fatalf("should get an error")
	}
}
