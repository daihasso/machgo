package qtypes

import (
    "fmt"
    "reflect"
    "strings"

    "github.com/pkg/errors"
    "github.com/jmoiron/sqlx"

    "github.com/daihasso/machgo/base"
    "github.com/daihasso/machgo/refl"
)


type AliasObjValMap map[string]*reflect.Value

// QueryResult is a set of all the results from a query in objects.
type QueryResult struct {
    rows *sqlx.Rows
    aliasedTables *AliasedTables
    aliasObjValPtr AliasObjValMap
    columnAliasFields []ColumnAliasField

    closeAfterWrite bool
}

func (self QueryResult) WriteTo(objects ...interface{}) error {
    aliasObjMap := make(AliasObjValMap, len(objects))
    for _, object := range objects {
        objValPtr := reflect.ValueOf(object)
        if objValPtr.Kind() != reflect.Ptr {
            return fmt.Errorf(
                "Object provided should be *%T not %T.",
                object,
                object,
            )
        }
        objVal := objValPtr.Elem()
        if objVal.Kind() == reflect.Ptr {
            objTypeStr := fmt.Sprintf("%T", object)
            baseType := strings.Replace(objTypeStr, "*", "", -1)
            return fmt.Errorf(
                "Object provided should be *%s not %s.",
                baseType,
                objTypeStr,
            )
        }

        objAlias, err := self.aliasedTables.ObjectAlias(object.(base.Base))
        if err != nil {
            return err
        }
        self.aliasObjValPtr[objAlias] = &objValPtr
        aliasObjMap[objAlias] = &objValPtr
    }

    return readRowIntoObjs(self.rows, aliasObjMap, self.columnAliasFields)
}

func (self QueryResult) Close() error {
    err := self.rows.Close()
    if err != nil {
        return errors.Wrap(err, "Error closing rows in QueryResult")
    }

    return nil
}

func readRowIntoObjs(
    rows *sqlx.Rows,
    aliasObjVals AliasObjValMap,
    columnAliasFields []ColumnAliasField,
) error {
    values := make([]interface{}, len(columnAliasFields))
    for i, columnAliasField := range columnAliasFields {
        objVal := aliasObjVals[columnAliasField.TableAlias]
        field := objVal.Elem().FieldByName(columnAliasField.FieldName)

        if !field.IsValid() {
            return fmt.Errorf(
                "Field in returned data '%s' is not valid.",
                columnAliasField.FieldName,
            )
        }
        // Stole from reflectx: https://tinyurl.com/yc3lpeam
        if field.Kind() == reflect.Ptr && field.IsNil() {
            alloc := reflect.New(refl.Deref(field.Type()))
            field.Set(alloc)
        }
        if field.Kind() == reflect.Map && field.IsNil() {
            field.Set(reflect.MakeMap(field.Type()))
        }

        values[i] = field.Addr().Interface()
    }

    return rows.Scan(values...)
}

func NewQueryResult(
    rows *sqlx.Rows,
    aliasedTables *AliasedTables,
    columnAliasFields []ColumnAliasField,
) (*QueryResult, error) {

    return &QueryResult{
        rows: rows,
        aliasedTables: aliasedTables,
        aliasObjValPtr: make(AliasObjValMap),
        columnAliasFields: columnAliasFields,
        closeAfterWrite: true,
    }, nil
}