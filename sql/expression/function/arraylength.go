package function // import "github.com/src-d/go-mysql-server/sql/expression/function"

import (
	"fmt"
	"reflect"

	"github.com/src-d/go-mysql-server/sql"
	"github.com/src-d/go-mysql-server/sql/expression"
)

// ArrayLength returns the length of an array.
type ArrayLength struct {
	expression.UnaryExpression
}

// NewArrayLength creates a new ArrayLength UDF.
func NewArrayLength(array sql.Expression) sql.Expression {
	return &ArrayLength{expression.UnaryExpression{Child: array}}
}

// Type implements the Expression interface.
func (*ArrayLength) Type() sql.Type {
	return sql.Int32
}

func (f *ArrayLength) String() string {
	return fmt.Sprintf("array_length(%s)", f.Child)
}

// TransformUp implements the Expression interface.
func (f *ArrayLength) TransformUp(fn sql.TransformExprFunc) (sql.Expression, error) {
	child, err := f.Child.TransformUp(fn)
	if err != nil {
		return nil, err
	}
	return fn(NewArrayLength(child))
}

// Eval implements the Expression interface.
func (f *ArrayLength) Eval(ctx *sql.Context, row sql.Row) (interface{}, error) {
	if t := f.Child.Type(); !sql.IsArray(t) && t != sql.JSON {
		return nil, sql.ErrInvalidType.New(f.Child.Type().Type().String())
	}

	child, err := f.Child.Eval(ctx, row)
	if err != nil {
		return nil, err
	}

	if child == nil {
		return nil, nil
	}

	array, ok := child.([]interface{})
	if !ok {
		return nil, sql.ErrInvalidType.New(reflect.TypeOf(child))
	}

	return int32(len(array)), nil
}
