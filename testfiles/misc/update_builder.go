package misc

import (
	"reflect"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/genproto/googleapis/type/latlng"
)

func isReservedType(value reflect.Value) bool {
	switch value.Interface().(type) {
	case time.Time, *time.Time,
		latlng.LatLng, *latlng.LatLng,
		firestore.DocumentRef, *firestore.DocumentRef:
		return true
	}
	return false
}

func updateBuilder(v, param interface{}) map[string]firestore.Update {
	rv := reflect.Indirect(reflect.ValueOf(v))
	rt := rv.Type()
	pv := reflect.Indirect(reflect.ValueOf(param))
	pt := pv.Type()
	updateMap := make(map[string]firestore.Update)
	for i := 0; i < rt.NumField(); i++ {
		ft := rt.Field(i)
		fv := rv.Field(i)

		if ft.Anonymous {
			for key, val := range updateBuilder(fv.Interface(), param) {
				if _, ok := updateMap[key]; ok {
					panic("fields with the same name cannot be used")
				}
				updateMap[key] = val
			}
			continue
		}

		if _, ok := pt.FieldByName(ft.Name); !ok {
			continue
		}

		path := ft.Name
		if firestoreTag, ok := ft.Tag.Lookup("firestore"); ok {
			path = strings.Split(firestoreTag, ",")[0]
		}

		pfv := pv.FieldByName(ft.Name)

		switch fv.Kind() {
		case reflect.Ptr:
			ptrType := reflect.PtrTo(fv.Type()).Elem()
			fv = reflect.New(ptrType.Elem())
			fallthrough
		case reflect.Struct:
			if isReservedType(fv) {
				break
			}
			for key, update := range updateBuilder(fv.Interface(), pfv.Interface()) {
				update.FieldPath = append(firestore.FieldPath{path}, update.FieldPath...)

				fp := make(firestore.FieldPath, len(update.FieldPath))
				copy(fp, update.FieldPath)

				sp := strings.Split(key, ".")
				fieldKey := strings.Join(append(fp[:len(update.FieldPath)-1], sp[len(sp)-1]), ".")

				if _, ok := updateMap[fieldKey]; ok {
					panic("fields with the same name cannot be used")
				}

				updateMap[fieldKey] = update
			}
			continue
		}

		if _, ok := updateMap[ft.Name]; ok {
			panic("fields with the same name cannot be used")
		}

		var isValid bool
		switch pfv.Kind() {
		case reflect.Interface, reflect.Ptr:
			if !pfv.IsNil() {
				isValid = true
			}
		default:
			if !pfv.IsZero() {
				isValid = true
			}
		}

		update := firestore.Update{FieldPath: firestore.FieldPath{path}}
		if isValid {
			update.Value = pfv.Interface()
		}

		if update.Value != nil {
			updateMap[ft.Name] = update
		}
	}

	return updateMap
}
