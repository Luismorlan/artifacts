package utils

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	"github.com/rnr-capital/newsfeed-backend/model"
	. "github.com/rnr-capital/newsfeed-backend/utils/log"
)

// TODO(jamie): optimize by first parsing json and match later
// TODO(jamie): should probably create a in-memory cache to avoid constant
// parsing the jsonStr into data expression because such kind of parsing is
// expensive.
func DataExpressionMatchPostChain(jsonStr string, rootPost *model.Post) (bool, error) {
	if len(jsonStr) == 0 {
		return true, nil
	}

	dataExpressionWrap, err := ParseDataExpression(jsonStr)
	if err != nil {
		LogV2.Errorf("data expression can't be unmarshaled to dataExpressionWrap, error :", err)
		return false, err
	}

	matched, err := DataExpressionMatch(dataExpressionWrap, rootPost)
	if err != nil {
		return false, errors.Wrap(err, "data expression match failed")
	}
	if matched {
		return true, nil
	}
	if rootPost.SharedFromPost != nil {
		return DataExpressionMatchPostChain(jsonStr, rootPost.SharedFromPost)
	}
	return false, nil
}

func ParseDataExpression(jsonStr string) (model.DataExpressionWrap, error) {
	emptyExpression := model.DataExpressionWrap{
		ID:   "",
		Expr: nil,
	}

	if len(jsonStr) == 0 {
		return emptyExpression, nil
	}

	var dataExpressionWrap model.DataExpressionWrap
	if err := json.Unmarshal([]byte(jsonStr), &dataExpressionWrap); err != nil {
		LogV2.Errorf("data expression can't be unmarshaled to dataExpressionWrap, error :", err)
		return emptyExpression, err
	}

	return dataExpressionWrap, nil
}

func DataExpressionToSql(dataExpressionWrap model.DataExpressionWrap) (string, error) {
	if dataExpressionWrap.IsEmpty() {
		return "True", nil
	}
	switch expr := dataExpressionWrap.Expr.(type) {
	case model.AllOf:
		res := "(TRUE "
		for _, child := range expr.AllOf {
			childSql, err := DataExpressionToSql(child)
			if err != nil {
				return "", errors.New("invalid data expression")
			}
			res += "AND (" + childSql + ")"
		}
		res += ")"
		return res, nil
	case model.AnyOf:
		res := "(FALSE "
		for _, child := range expr.AnyOf {
			childSql, err := DataExpressionToSql(child)
			if err != nil {
				return "", errors.New("invalid data expression")
			}
			res += "OR (" + childSql + ")"
		}
		res += ")"
		return res, nil
	case model.NotTrue:
		res := "(NOT ("
		childSql, err := DataExpressionToSql(expr.NotTrue)
		if err != nil {
			return "", errors.New("invalid data expression")
		}
		res += childSql + "))"
		return res, nil
	case model.PredicateWrap:
		if expr.Predicate.Type == "LITERAL" {
			// we created a gin index on the function of concat(title and content)
			// notice the order matters so title has to be in front of content
			// the text search should be close to an O(1) index search
			return "coalesce((shared_from_post.title || '' || shared_from_post.content) ILIKE '%" +
				expr.Predicate.Param.Text +
				"%', false) OR (posts.title || '' || posts.content) ILIKE '%" +
				expr.Predicate.Param.Text + "%'", nil
		}
		return "", errors.New("invalid literal")
	default:
		return "", errors.New("invalid data expression")
	}
}

func DataExpressionMatch(dataExpressionWrap model.DataExpressionWrap, post *model.Post) (bool, error) {
	// Empty data expression should match all post.
	if dataExpressionWrap.IsEmpty() {
		return true, nil
	}
	switch expr := dataExpressionWrap.Expr.(type) {
	case model.AllOf:
		if len(expr.AllOf) == 0 {
			return true, nil
		}
		for _, child := range expr.AllOf {
			match, err := DataExpressionMatch(child, post)
			if err != nil {
				return false, err
			}
			if !match {
				return false, nil
			}
		}
		return true, nil
	case model.AnyOf:
		if len(expr.AnyOf) == 0 {
			return true, nil
		}
		for _, child := range expr.AnyOf {
			match, err := DataExpressionMatch(child, post)
			if err != nil {
				return false, err
			}
			if match {
				return true, nil
			}
		}
		return false, nil
	case model.NotTrue:
		match, err := DataExpressionMatch(expr.NotTrue, post)
		if err != nil {
			return false, err
		}
		return !match, nil
	case model.PredicateWrap:
		if expr.Predicate.Type == "LITERAL" {
			return strings.Contains(strings.ToLower(post.Content), strings.ToLower(expr.Predicate.Param.Text)) || strings.Contains(strings.ToLower(post.Title), strings.ToLower(expr.Predicate.Param.Text)), nil
		}
	default:
		return false, errors.New("unknown node type when matching data expression")
	}
	return false, nil
}
