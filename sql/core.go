// Copyright 2020-2021 Dolthub, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sql

import (
	"fmt"
	"math"
	"strconv"
	"time"
)

// Nameable is something that has a name.
type Nameable interface {
	// Name returns the name.
	Name() string
}

// Tableable is something that has a table.
type Tableable interface {
	// Table returns the table name.
	Table() string
}

// Resolvable is something that can be resolved or not.
type Resolvable interface {
	// Resolved returns whether the node is resolved.
	Resolved() bool
}

// BinaryNode has two children
type BinaryNode interface {
	Left() Node
	Right() Node
}

// UnaryNode has one child
type UnaryNode interface {
	Child() Node
}

// Expression is a combination of one or more SQL expressions.
type Expression interface {
	Resolvable
	fmt.Stringer
	// Type returns the expression type.
	Type() Type
	// IsNullable returns whether the expression can be null.
	IsNullable() bool
	// Eval evaluates the given row and returns a result.
	Eval(ctx *Context, row Row) (interface{}, error)
	// Children returns the children expressions of this expression.
	Children() []Expression
	// WithChildren returns a copy of the expression with children replaced.
	// It will return an error if the number of children is different than
	// the current number of children. They must be given in the same order
	// as they are returned by Children.
	WithChildren(children ...Expression) (Expression, error)
}

type Expression2 interface {
	Expression
	// Eval2 evaluates the given row frame and returns a result.
	Eval2(ctx *Context, row Row2) (Value, error)
	// Type2 returns the expression type.
	Type2() Type2
}

// UnsupportedFunctionStub is a marker interface for function stubs that are unsupported
type UnsupportedFunctionStub interface {
	IsUnsupported() bool
}

// FunctionExpression is an Expression that represents a function.
type FunctionExpression interface {
	Expression
	FunctionName() string
	Description() string
	// TODO: add Example() function
}

// NonDeterministicExpression allows a way for expressions to declare that they are non-deterministic, which will
// signal the engine to not cache their results when this would otherwise appear to be safe.
type NonDeterministicExpression interface {
	Expression
	// IsNonDeterministic returns whether this expression returns a non-deterministic result. An expression is
	// non-deterministic if it can return different results on subsequent evaluations.
	IsNonDeterministic() bool
}

// Node is a node in the execution plan tree.
type Node interface {
	Resolvable
	fmt.Stringer
	// Schema of the node.
	Schema() Schema
	// Children nodes.
	Children() []Node
	// RowIter produces a row iterator from this node. The current row being evaluated is provided, as well the context
	// of the query.
	RowIter(ctx *Context, row Row) (RowIter, error)
	// WithChildren returns a copy of the node with children replaced.
	// It will return an error if the number of children is different than
	// the current number of children. They must be given in the same order
	// as they are returned by Children.
	WithChildren(children ...Node) (Node, error)
	// CheckPrivileges passes the operations representative of this Node to the PrivilegedOperationChecker to determine
	// whether a user (contained in the context, along with their active roles) has the necessary privileges to execute
	// this node (and its children).
	CheckPrivileges(ctx *Context, opChecker PrivilegedOperationChecker) bool
}

type Node2 interface {
	Node

	// RowIter2 produces a row iterator from this node. The current row frame being
	// evaluated is provided, as well the context of the query.
	RowIter2(ctx *Context, f *RowFrame) (RowIter2, error)
}

// RowIterTypeSelector is implemented by top-level type-switch nodes that return either a Node or Node2 implementation.
type RowIterTypeSelector interface {
	RowIter
	IsNode2() bool
}

// CommentedNode allows comments to be set and retrieved on it
type CommentedNode interface {
	Node
	WithComment(string) Node
	Comment() string
}

// DebugStringer is shared by implementors of Node and Expression, and is used for debugging the analyzer. It allows
// a node or expression to be printed in greater detail than its default String() representation.
type DebugStringer interface {
	// DebugString prints a debug string of the node in question.
	DebugString() string
}

// DebugString returns a debug string for the Node or Expression given.
func DebugString(nodeOrExpression interface{}) string {
	if ds, ok := nodeOrExpression.(DebugStringer); ok {
		return ds.DebugString()
	}
	if s, ok := nodeOrExpression.(fmt.Stringer); ok {
		return s.String()
	}
	panic(fmt.Sprintf("Expected sql.DebugString or fmt.Stringer for %T", nodeOrExpression))
}

// OpaqueNode is a node that doesn't allow transformations to its children and
// acts as a black box.
type OpaqueNode interface {
	Node
	// Opaque reports whether the node is opaque or not.
	Opaque() bool
}

// Projector is a node that projects expressions for parent nodes to consume (i.e. GroupBy, Window, Project).
type Projector interface {
	// ProjectedExprs returns the list of expressions projected by this node.
	ProjectedExprs() []Expression
	// WithProjectedExprs returns a new Projector instance with the specified expressions set as its projected expressions.
	WithProjectedExprs(...Expression) (Projector, error)
}

// Expressioner is a node that contains expressions.
type Expressioner interface {
	// Expressions returns the list of expressions contained by the node.
	Expressions() []Expression
	// WithExpressions returns a copy of the node with expressions replaced.
	// It will return an error if the number of expressions is different than
	// the current number of expressions. They must be given in the same order
	// as they are returned by Expressions.
	WithExpressions(...Expression) (Node, error)
}

// SchemaTarget is a node that has a target schema that can be set during analysis. This is necessary because some
// schema objects (things that involve expressions, column references, etc.) can only be reified during analysis. The
// target schema is the schema of a table under a DDL operation, not the schema of rows returned by this node.
type SchemaTarget interface {
	// WithTargetSchema returns a copy of this node with the target schema set
	WithTargetSchema(Schema) (Node, error)
	// TargetSchema returns the target schema for this node
	TargetSchema() Schema
}

// PrimaryKeySchemaTarget is a node that has a primary key target schema that can be set
type PrimaryKeySchemaTarget interface {
	SchemaTarget
	WithPrimaryKeySchema(schema PrimaryKeySchema) (Node, error)
}

// PartitionCounter can return the number of partitions.
type PartitionCounter interface {
	// PartitionCount returns the number of partitions.
	PartitionCount(*Context) (int64, error)
}

// ForeignKeyEditor is a TableEditor that is addressable via IndexLookup.
type ForeignKeyEditor interface {
	TableEditor
	IndexAddressable
}

type Closer interface {
	Close(*Context) error
}

// ExternalStoredProcedureProvider provides access to built-in stored procedures. These procedures are implemented
// as functions, instead of as SQL statements. The returned stored procedures cannot be modified or deleted.
type ExternalStoredProcedureProvider interface {
	// ExternalStoredProcedure returns the external stored procedure details for the procedure with the specified name
	// that is able to accept the specified number of parameters. If no matching external stored procedure is found,
	// nil, nil is returned. If an unexpected error is encountered, it is returned as the error parameter.
	ExternalStoredProcedure(ctx *Context, name string, numOfParams int) (*ExternalStoredProcedureDetails, error)
	// ExternalStoredProcedures returns a slice of all external stored procedure details with the specified name. External
	// stored procedures can overload the same name with different arguments, so this method enables a caller to see all
	// available variants with the specified name. If no matching external stored procedures are found, an
	// empty slice is returned, with a nil error. If an unexpected error is encountered, it is returned as the
	// error parameter.
	ExternalStoredProcedures(ctx *Context, name string) ([]ExternalStoredProcedureDetails, error)
}

// FunctionProvider is an extension of DatabaseProvider that allows custom functions to be provided
type FunctionProvider interface {
	// Function returns the function with the name provided, case-insensitive
	Function(ctx *Context, name string) (Function, error)
}

// TableFunctionProvider is an extension of DatabaseProvider that allows custom table functions to be provided
type TableFunctionProvider interface {
	// TableFunction returns the table function with the name provided, case-insensitive
	TableFunction(ctx *Context, name string) (TableFunction, error)
}

type TransactionCharacteristic int

const (
	ReadWrite TransactionCharacteristic = iota
	ReadOnly
)

// Transaction is an opaque type implemented by an integrator to record necessary information at the start of a
// transaction. Active transactions will be recorded in the session.
type Transaction interface {
	fmt.Stringer
	IsReadOnly() bool
}

// TriggerDefinition defines a trigger. Integrators are not expected to parse or understand the trigger definitions,
// but must store and return them when asked.
type TriggerDefinition struct {
	Name            string    // The name of this trigger. Trigger names in a database are unique.
	CreateStatement string    // The text of the statement to create this trigger.
	CreatedAt       time.Time // The time that the trigger was created.
}

// ViewDefinition is the named textual definition of a view
type ViewDefinition struct {
	Name           string
	TextDefinition string
}

// ViewDatabase is implemented by databases that persist view definitions
type ViewDatabase interface {
	// CreateView persists the definition a view with the name and select statement given. If a view with that name
	// already exists, should return ErrExistingView
	CreateView(ctx *Context, name string, selectStatement string) error

	// DropView deletes the view named from persistent storage. If the view doesn't exist, should return
	// ErrViewDoesNotExist
	DropView(ctx *Context, name string) error

	// GetView returns the textual definition of the view with the name given, or false if it doesn't exist.
	GetView(ctx *Context, viewName string) (string, bool, error)

	// AllViews returns the definitions of all views in the database
	AllViews(ctx *Context) ([]ViewDefinition, error)
}

// ColumnOrder is used in ALTER TABLE statements to change the order of inserted / modified columns.
type ColumnOrder struct {
	First       bool   // True if this column should come first
	AfterColumn string // Set to the name of the column after which this column should appear
}

// Lockable should be implemented by tables that can be locked and unlocked.
type Lockable interface {
	Nameable
	// Lock locks the table either for reads or writes. Any session clients can
	// read while the table is locked for read, but not write.
	// When the table is locked for write, nobody can write except for the
	// session client that requested the lock.
	Lock(ctx *Context, write bool) error
	// Unlock releases the lock for the current session client. It blocks until
	// all reads or writes started during the lock are finished.
	// Context may be nil if the unlock it's because the connection was closed.
	// The id will always be provided, since in some cases context is not
	// available.
	Unlock(ctx *Context, id uint32) error
}

// StoredProcedureDetails are the details of the stored procedure. Integrators only need to store and retrieve the given
// details for a stored procedure, as the engine handles all parsing and processing.
type StoredProcedureDetails struct {
	Name            string    // The name of this stored procedure. Names must be unique within a database.
	CreateStatement string    // The CREATE statement for this stored procedure.
	CreatedAt       time.Time // The time that the stored procedure was created.
	ModifiedAt      time.Time // The time of the last modification to the stored procedure.
}

// ExternalStoredProcedureDetails are the details of an external stored procedure. Compared to standard stored
// procedures, external ones are considered "built-in", in that they're not created by the user, and may not be modified
// or deleted by a user. In addition, they're implemented as a function taking standard parameters, compared to stored
// procedures being implemented as expressions.
type ExternalStoredProcedureDetails struct {
	// Name is the name of the external stored procedure. If two external stored procedures share a name, then they're
	// considered overloaded. Standard stored procedures do not support overloading.
	Name string
	// Schema describes the row layout of the RowIter returned from Function.
	Schema Schema
	// Function is the implementation of the external stored procedure. All functions should have the following definition:
	// `func(*Context, <PARAMETERS>) (RowIter, error)`. The <PARAMETERS> may be any of the following types: `bool`,
	// `string`, `[]byte`, `int8`-`int64`, `uint8`-`uint64`, `float32`, `float64`, `time.Time`, or `Decimal`
	// (shopspring/decimal). The architecture-dependent types `int` and `uint` (without a number) are also supported.
	// It is valid to return a nil RowIter if there are no rows to be returned.
	//
	// Each parameter, by default, is an IN parameter. If the parameter type is a pointer, e.g. `*int32`, then it
	// becomes an INOUT parameter. INOUT parameters will be given their zero value if the parameter's value is nil.
	// There is no way to set a parameter as an OUT parameter.
	//
	// Values are converted to their nearest type before being passed in, following the conversion rules of their
	// related SQL types. The exceptions are `time.Time` (treated as a `DATETIME`), string (treated as a `LONGTEXT` with
	// the default collation) and Decimal (treated with a larger precision and scale). Take extra care when using decimal
	// for an INOUT parameter, to ensure that the returned value fits the original's precision and scale, else an error
	// will occur.
	//
	// As functions support overloading, each variant must have a completely unique function signature to prevent
	// ambiguity. Uniqueness is determined by the number of parameters. If two functions are returned that have the same
	// name and same number of parameters, then an error is thrown. If the last parameter is variadic, then the stored
	// procedure functions as though it has the integer-max number of parameters. When an exact match is not found for
	// overloaded functions, the largest function is used (which in this case will be the variadic function). Also, due
	// to the usage of the integer-max for the parameter count, only one variadic function is allowed per function name.
	// The type of the variadic parameter may not have a pointer type.
	Function interface{}
}

// FakeCreateProcedureStmt returns a parseable CREATE PROCEDURE statement for this external stored procedure, as some
// tools (such as Java's JDBC connector) require a valid statement in some situations.
func (espd ExternalStoredProcedureDetails) FakeCreateProcedureStmt() string {
	return fmt.Sprintf("CREATE PROCEDURE %s() SELECT 'External stored procedure';", espd.Name)
}

// EvaluateCondition evaluates a condition, which is an expression whose value
// will be nil or coerced boolean.
func EvaluateCondition(ctx *Context, cond Expression, row Row) (interface{}, error) {
	v, err := cond.Eval(ctx, row)
	if err != nil {
		return false, err
	}
	if v == nil {
		return nil, nil
	}

	switch b := v.(type) {
	case bool:
		return b, nil
	case int:
		return b != int(0), nil
	case int64:
		return b != int64(0), nil
	case int32:
		return b != int32(0), nil
	case int16:
		return b != int16(0), nil
	case int8:
		return b != int8(0), nil
	case uint:
		return b != uint(0), nil
	case uint64:
		return b != uint64(0), nil
	case uint32:
		return b != uint32(0), nil
	case uint16:
		return b != uint16(0), nil
	case uint8:
		return b != uint8(0), nil
	case time.Duration:
		return int64(b) != 0, nil
	case time.Time:
		return b.UnixNano() != 0, nil
	case float64:
		return int(math.Round(v.(float64))) != 0, nil
	case float32:
		return int(math.Round(float64(v.(float32)))) != 0, nil
	case string:
		parsed, err := strconv.ParseFloat(v.(string), 64)
		return err == nil && int(parsed) != 0, nil
	default:
		return false, nil
	}
}

// IsFalse coerces EvaluateCondition interface{} response to boolean
func IsFalse(val interface{}) bool {
	res, ok := val.(bool)
	return ok && !res
}

// IsTrue coerces EvaluateCondition interface{} response to boolean
func IsTrue(val interface{}) bool {
	res, ok := val.(bool)
	return ok && res
}

// TypesEqual compares two Types and returns whether they are equivalent.
func TypesEqual(a, b Type) bool {
	// TODO: replace all of the Type() == Type() calls with TypesEqual

	// We can assume they have the same implementing type if this passes, so we have to check the parameters
	if a.Type() != b.Type() {
		return false
	}
	// Some types cannot be compared structurally as they contain non-comparable types (such as slices), so we handle
	// those separately.
	switch at := a.(type) {
	case enumType:
		aEnumType := at
		bEnumType := b.(enumType)
		if len(aEnumType.indexToVal) != len(bEnumType.indexToVal) {
			return false
		}
		for i := 0; i < len(aEnumType.indexToVal); i++ {
			if aEnumType.indexToVal[i] != bEnumType.indexToVal[i] {
				return false
			}
		}
		return aEnumType.collation == bEnumType.collation
	case setType:
		aSetType := at
		bSetType := b.(setType)
		if len(aSetType.bitToVal) != len(bSetType.bitToVal) {
			return false
		}
		for bit, aVal := range aSetType.bitToVal {
			if bVal, ok := bSetType.bitToVal[bit]; ok && aVal != bVal {
				return false
			}
		}
		return aSetType.collation == bSetType.collation
	case TupleType:
		if tupA, ok := a.(TupleType); ok {
			if tupB, ok := b.(TupleType); ok && len(tupA) == len(tupB) {
				for i := range tupA {
					if !TypesEqual(tupA[i], tupB[i]) {
						return false
					}
				}
				return true
			}
		}
		return false
	default:
		return a == b
	}
}
