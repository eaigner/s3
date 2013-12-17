package s3

import (
	"time"
)

type Policy map[string]interface{}

func (p Policy) SetExpiration(seconds uint) {
	p["expiration"] = time.Now().UTC().Add(time.Second * time.Duration(seconds)).Format("2006-01-02T15:04:05Z")
}

func (p Policy) Conditions() *PolicyConditions {
	key := "conditions"
	if _, ok := p[key]; !ok {
		pol := make(PolicyConditions, 0, 5)
		p[key] = &pol
	}
	if t, ok := p[key].(*PolicyConditions); ok {
		return t
	}
	panic("unreachable")
}

type PolicyConditions []interface{}

func (c *PolicyConditions) Add(key, value string) {
	*c = append(*c, map[string]string{key: value})
}

func (c *PolicyConditions) AddBucket(bucket string) {
	c.Add("bucket", bucket)
}

func (c *PolicyConditions) AddACL(acl ACL) {
	c.Add("acl", string(acl))
}

func (c *PolicyConditions) AddRedirect(url string) {
	c.Add("redirect", url)
}

func (c *PolicyConditions) AddSuccessActionRedirect(url string) {
	c.Add("success_action_redirect", url)
}

func (c *PolicyConditions) Match(mtype, cond, match string) {
	*c = append(*c, []string{mtype, cond, match})
}

func (c *PolicyConditions) MatchEquals(cond, match string) {
	c.Match("eq", cond, match)
}

func (c *PolicyConditions) MatchStartsWith(cond, match string) {
	c.Match("starts-with", cond, match)
}
