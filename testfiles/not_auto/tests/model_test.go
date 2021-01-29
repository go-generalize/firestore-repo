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

	now := time.Unix(0, time.Now().UnixNano())
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
					tr2.Log(sub.ID, sts[1].ID)
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
		err := client.RunTransaction(ctx, func(cx context.Context, tx *firestore.Transaction) error {
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
		})

		if err != nil {
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
		err := client.RunTransaction(ctx, func(cx context.Context, tx *firestore.Transaction) error {
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
		})

		if err != nil {
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

	tks := make([]*model.Task, 0)
	latLng := &latlng.LatLng{
		Latitude:  35.678803,
		Longitude: 139.756263,
	}
	for i := 1; i <= 10; i++ {
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

	t.Run("Geo(10件)", func(t *testing.T) {
		req := &model.TaskListReq{
			Geo: model.NewQueryChainer().Equal(latLng),
		}

		tasks, err := taskRepo.List(ctx, req, nil)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		if len(tasks) != 10 {
			t.Fatal("not match")
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
