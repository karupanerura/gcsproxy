package gcsproxy

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

func parseCacheCondition(header http.Header) (cacheCondition, error) {
	cond := cacheCondition{}
	if ifMatch, ifNoneMatch := header.Get("If-Match"), header.Get("If-None-Match"); ifMatch != "" || ifNoneMatch != "" {
		if ifMatch != "" && ifNoneMatch != "" {
			return cacheCondition{}, fmt.Errorf("conflict If-Match and If-None-Match headers")
		}

		if ifMatch != "" {
			values, isWildcard, err := parseETagValuesToSet(ifMatch)
			if err != nil {
				return cacheCondition{}, fmt.Errorf("invalid If-Match: %w", err)
			}

			cond.ETag = &eTagCondition{
				Policy:     ifMatchPolicy,
				Values:     values,
				IsWildcard: isWildcard,
			}
		}
		if ifNoneMatch != "" {
			values, isWildcard, err := parseETagValuesToSet(ifNoneMatch)
			if err != nil {
				return cacheCondition{}, fmt.Errorf("invalid If-None-Match: %w", err)
			}

			cond.ETag = &eTagCondition{
				Policy:     ifNoneMatchPolicy,
				Values:     values,
				IsWildcard: isWildcard,
			}
		}
	}
	if ifModifiedSince, ifUnmodifiedSince := header.Get("If-Modified-Since"), header.Get("If-Unmodified-Since"); ifModifiedSince != "" || ifUnmodifiedSince != "" {
		if ifModifiedSince != "" && ifUnmodifiedSince != "" {
			return cacheCondition{}, fmt.Errorf("conflict If-Modified-Since and If-Unmodified-Since headers")
		}

		if ifModifiedSince != "" {
			t, err := time.Parse(time.RFC1123, ifModifiedSince)
			if err != nil {
				return cacheCondition{}, fmt.Errorf("invalid If-Modified-Since: %w", err)
			}

			cond.Time = &timeCondition{
				Policy: ifModifiedSincePolicy,
				Value:  t,
			}
		}
		if ifUnmodifiedSince != "" {
			t, err := time.Parse(time.RFC1123, ifUnmodifiedSince)
			if err != nil {
				return cacheCondition{}, fmt.Errorf("invalid If-Unmodified-Since: %w", err)
			}

			cond.Time = &timeCondition{
				Policy: ifUnmodifiedSincePolicy,
				Value:  t,
			}
		}
	}

	return cond, nil
}

type cacheCondition struct {
	Time *timeCondition
	ETag *eTagCondition
}

type timeConditionPolicy int

const (
	ifModifiedSincePolicy timeConditionPolicy = iota
	ifUnmodifiedSincePolicy
)

type timeCondition struct {
	Policy timeConditionPolicy
	Value  time.Time
}

func (c *timeCondition) Match(lastModified time.Time) bool {
	switch c.Policy {
	case ifModifiedSincePolicy:
		return c.Value.Before(lastModified)

	case ifUnmodifiedSincePolicy:
		return !c.Value.Before(lastModified)
	}
	panic("should not reach here")
}

type eTagConditionPolicy int

const (
	ifMatchPolicy eTagConditionPolicy = iota
	ifNoneMatchPolicy
)

type eTagCondition struct {
	Policy     eTagConditionPolicy
	Values     map[string]eTagType
	IsWildcard bool
}

type eTagType struct {
	IsWeak bool
}

func (c *eTagCondition) Match(eTag string, isWeak bool) bool {
	switch c.Policy {
	case ifNoneMatchPolicy:
		if c.IsWildcard {
			return false
		}
		_, ok := c.Values[eTag]
		return !ok

	case ifMatchPolicy:
		if c.IsWildcard {
			return true
		}
		typ, ok := c.Values[eTag]
		return ok && isWeak == typ.IsWeak
	}
	panic("should not reach here")
}

func parseETagValuesToSet(eTag string) (map[string]eTagType, bool, error) {
	eTag = strings.TrimSpace(eTag)
	if eTag == "*" {
		return nil, true, nil
	}

	m := map[string]eTagType{}
	offset := 0
	for i := strings.IndexAny(eTag[offset:], `W"`); i != -1; i = strings.IndexAny(eTag[offset:], `W",`) {
		weakRef := false
		switch eTag[offset+i] {
		case 'W':
			if offset+i+2 >= len(eTag) || !strings.HasPrefix(eTag[offset+i:], `W/"`) {
				return nil, false, fmt.Errorf("x invalid character at %d", i)
			}
			offset += 2
			weakRef = true
			fallthrough
		case '"':
			fmt.Println(eTag)
			fmt.Println(offset)
			fmt.Println(i)
			fmt.Println(len(eTag))
			if offset+i+1 >= len(eTag) {
				return nil, false, fmt.Errorf("y invalid character at %d", i)
			}

			e := strings.IndexByte(eTag[offset+i+1:], '"')
			if e == -1 {
				return nil, false, fmt.Errorf("z invalid character at %d", i)
			}

			t := eTag[offset+i+1 : offset+i+e+1]
			m[t] = eTagType{IsWeak: weakRef}
			offset += i + 1 + e + 1 + strings.IndexAny(eTag[offset+i+1+e+1:], " \t") + 1
		case ',':
			offset += i + 1 + strings.IndexAny(eTag[offset+i+1:], " \t") + 1
		}
	}
	t := strings.TrimSpace(eTag[offset:])
	if t != "" {
		return nil, false, fmt.Errorf("invalid character at %d", offset)
	}

	return m, false, nil
}
