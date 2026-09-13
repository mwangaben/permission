package storage

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

const (
	DriverGorm = "gorm"
	DriverEnt  = "ent"
)

// DetectDriver inspects an opaque DB handle and returns the driver name.
//
// Supported inputs:
//   - *gorm.DB
//   - *ent.Client  (from this module's storage/entstore/ent package)
//   - any value exposing DB() *gorm.DB
//   - any value exposing Client() *ent.Client
func DetectDriver(db interface{}) (string, error) {
	if db == nil {
		return "", errors.New("storage: nil db")
	}

	t := reflect.TypeOf(db)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	pkgPath := t.PkgPath()
	typeName := t.Name()

	switch {
	case pkgPath == "gorm.io/gorm" && typeName == "DB":
		return DriverGorm, nil

	// Ent's generated client lives in a package whose path ends with "/ent".
	case typeName == "Client" && strings.HasSuffix(pkgPath, "/ent"):
		return DriverEnt, nil

	case hasMethod(db, "Client"):
		return DriverEnt, nil
	case hasMethod(db, "DB"):
		return DriverGorm, nil
	}

	return "", fmt.Errorf("storage: unsupported db type %s.%s", pkgPath, typeName)
}

func hasMethod(v interface{}, name string) bool {
	_, ok := reflect.TypeOf(v).MethodByName(name)
	return ok
}
