package mysql

import (
	"fmt"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/db"

	"github.com/pingcap/tidb/parser/ast"
)

var (
	_ advisor.Advisor = (*WhereRequirementAdvisor)(nil)
)

func init() {
	advisor.Register(db.MySQL, advisor.MySQLWhereRequirement, &WhereRequirementAdvisor{})
	advisor.Register(db.TiDB, advisor.MySQLWhereRequirement, &WhereRequirementAdvisor{})
}

// WhereRequirementAdvisor is the advisor checking for the WHERE clause requirement for UPDATE/DELETE.
type WhereRequirementAdvisor struct {
}

// Check checks for the WHERE clause requirement for UPDATE/DELETE.
func (adv *WhereRequirementAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	root, errAdvice := parseStatement(statement, ctx.Charset, ctx.Collation)
	if errAdvice != nil {
		return errAdvice, nil
	}

	level, err := advisor.NewStatusBySchemaReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return []advisor.Advice{}, err
	}
	we := &whereRequirementChecker{level: level}
	for _, stmtNode := range root {
		(stmtNode).Accept(we)
	}

	if len(we.advisorList) == 0 {
		we.advisorList = append(we.advisorList, advisor.Advice{
			Status:  advisor.Success,
			Code:    common.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return we.advisorList, nil
}

type whereRequirementChecker struct {
	advisorList []advisor.Advice
	level       advisor.Status
}

func (v *whereRequirementChecker) Enter(in ast.Node) (ast.Node, bool) {
	code := common.Ok
	switch node := in.(type) {
	// DELETE
	case *ast.DeleteStmt:
		if node.Where == nil {
			code = common.StatementNoWhere
		}
	// UPDATE
	case *ast.UpdateStmt:
		if node.Where == nil {
			code = common.StatementNoWhere
		}
	}

	if code != common.Ok {
		v.advisorList = append(v.advisorList, advisor.Advice{
			Status:  v.level,
			Code:    code,
			Title:   "Require WHERE clause",
			Content: fmt.Sprintf("%q requires WHERE clause", in.Text()),
		})
	}
	return in, false
}

func (v *whereRequirementChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
