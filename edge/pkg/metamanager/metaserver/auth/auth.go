package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/square/go-jose.v2/jwt"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/serviceaccount"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

type jwtTokenAuthenticator struct {
	// TODO implement cache indexer to authenticate token
	indexer      cache.Indexer
	issuers      map[string]bool
	keys         []interface{}
	validator    serviceaccount.Validator
	implicitAuds authenticator.Audiences
}

func JWTTokenAuthenticator(indexer cache.Indexer, issuers []string, keys []interface{}, implicitAuds authenticator.Audiences, validator serviceaccount.Validator) authenticator.Token {
	issuersMap := make(map[string]bool)
	for _, issuer := range issuers {
		issuersMap[issuer] = true
	}
	return &jwtTokenAuthenticator{
		indexer:      indexer,
		issuers:      issuersMap,
		keys:         keys,
		implicitAuds: implicitAuds,
		validator:    validator,
	}
}

func (j *jwtTokenAuthenticator) hasCorrectIssuer(tokenData string) bool {
	parts := strings.Split(tokenData, ".")
	if len(parts) != 3 {
		return false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	claims := struct {
		// WARNING: this JWT is not verified. Do not trust these claims.
		Issuer string `json:"iss"`
	}{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return false
	}
	return j.issuers[claims.Issuer]
}

func (j *jwtTokenAuthenticator) AuthenticateToken(ctx context.Context, tokenData string) (*authenticator.Response, bool, error) {
	if !j.hasCorrectIssuer(tokenData) {
		return nil, false, nil
	}
	public := &jwt.Claims{}
	private := j.validator.NewPrivateClaims()
	if err := parseSigned(tokenData, public, private); err != nil {
		return nil, false, err
	}
	if len(j.keys) == 0 {
		// no public key for decode, auth token is existing in local db
		if !client.CheckTokenExist(tokenData) {
			return nil, false, fmt.Errorf("tokenData not found when authenticating")
		}
	} else {
		tok, err := jwt.ParseSigned(tokenData)
		if err != nil {
			return nil, false, nil
		}
		var (
			found   bool
			errlist []error
		)
		for _, key := range j.keys {
			if err := tok.Claims(key, public, private); err != nil {
				errlist = append(errlist, err)
				continue
			}
			found = true
			break
		}

		if !found {
			return nil, false, utilerrors.NewAggregate(errlist)
		}
	}

	tokenAudiences := authenticator.Audiences(public.Audience)
	if len(tokenAudiences) == 0 {
		tokenAudiences = j.implicitAuds
	}

	requestedAudiences, ok := authenticator.AudiencesFrom(ctx)
	if !ok {
		// default to apiserver audiences
		requestedAudiences = j.implicitAuds
	}

	auds := authenticator.Audiences(tokenAudiences).Intersect(requestedAudiences)
	if len(auds) == 0 && len(j.implicitAuds) != 0 {
		return nil, false, fmt.Errorf("tokenData audiences %q is invalid for the target audiences %q", tokenAudiences, requestedAudiences)
	}

	// If we get here, we have a tokenData with a recognized signature and
	// issuer string.
	sa, err := j.validator.Validate(ctx, tokenData, public, private)
	if err != nil {
		return nil, false, err
	}

	return &authenticator.Response{
		User:      sa.UserInfo(),
		Audiences: auds,
	}, true, nil
}
