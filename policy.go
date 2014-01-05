package s3

import (
	"time"
)

type (
	Policy    policyMap
	policyMap map[string]interface{}
)

func (p Policy) SetExpiration(seconds uint) {
	exp := time.Now().UTC().Add(time.Second * time.Duration(seconds))
	p["expiration"] = exp.Format("2006-01-02T15:04:05Z")
}

func (p Policy) Conditions() *PolicyConditions {
	key := "conditions"
	v, ok := p[key]
	if !ok {
		c := make(PolicyConditions, 0, 5)
		v, p[key] = &c, &c
	}
	return v.(*PolicyConditions)
}

type PolicyConditions []interface{}

func (c *PolicyConditions) Bucket(bucket string) {
	c.addKv("bucket", bucket)
}

func (c *PolicyConditions) ACL(acl ACL) {
	c.addKv("acl", string(acl))
}

func (c *PolicyConditions) Redirect(url string) {
	c.addKv("redirect", url)
}

func (c *PolicyConditions) SuccessActionRedirect(url string) {
	c.addKv("success_action_redirect", url)
}

func (c *PolicyConditions) Equals(cond, match string) {
	c.addArray("eq", cond, match)
}

func (c *PolicyConditions) StartsWith(cond, match string) {
	c.addArray("starts-with", cond, match)
}

func (c *PolicyConditions) ContentLengthRange(from, to int) {
	c.addArray("content-length-range", from, to)
}

// private

func (c *PolicyConditions) addKv(key, value string) {
	*c = append(*c, map[string]string{key: value})
}

func (c *PolicyConditions) addArray(key string, args ...interface{}) {
	*c = append(*c, append([]interface{}{key}, args...))
}
