package sess

import (
	"fmt"
	"reflect"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

	"MachGo/base"
)

var getObjectStatementTemplate = `SELECT * FROM %s WHERE %s`

func getObject(
	target base.Base, idValue interface{}, session *Session,
) error {
	var err error

	identifier := identifierFromBase(target)
	if !identifier.exists {
		return errors.New(
			"Object provided to GetObject doesn't have an identifier.",
		)
	} else if identifier.isSet {
		return errors.New(
			"Object provided to GetObject has an identifier set, it should " +
				"be a new instance with no identifier.",
		)
	}

	if identifier.value != nil &&
		reflect.TypeOf(identifier.value) != reflect.TypeOf(idValue) {
		return errors.Errorf(
			"Type of provided id (%T) does not match identifier type for " +
				"object (%T).",
			idValue,
			identifier.value,
		)
	}

	idColumn := objectIdColumn(target)

	tableName, err := base.BaseTable(target)
	if err != nil {
		return errors.Wrap(err, "Error while trying to get table name")
	}

	whereClause := fmt.Sprintf("%s = @%s", idColumn, idColumn)

	statement := fmt.Sprintf(
		getObjectStatementTemplate, tableName, whereClause,
	)

	err = session.Transactionized(func(tx *sqlx.Tx) error {
		var err error
		statement = tx.Rebind(statement)

		row := tx.QueryRowx(statement, idValue)

		err = row.StructScan(target)
		if err != nil {
			return errors.Wrap(
				err, "Error while reading data from DB",
			)
		}

		return nil
	})
	if err != nil {
		return err
	}

	err = setObjectSaved(target)
	if err != nil {
		return err
	}

	return nil
}

func GetObject(object base.Base, idValue interface{}) error {
	session, err := NewSessionFromGlobal()
	if err != nil {
		return errors.Wrap(
			err, "Couldn't get session from global connection pool",
		)
	}

	return getObject(object, idValue, session)
}