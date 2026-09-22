package handler

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
)

func TestContentModerationV2ServerEventIdentity(t *testing.T) {
	makeContext := func() *gin.Context {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", "/v1/responses", nil)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxkey.RequestID, "client-chosen"))
		return c
	}
	c := makeContext()
	first := buildContentModerationInput(c, nil, middleware2.AuthSubject{}, service.ContentModerationProtocolOpenAIResponses, "", nil)
	same := buildContentModerationInput(c, nil, middleware2.AuthSubject{}, service.ContentModerationProtocolOpenAIResponses, "", nil)
	another := buildContentModerationInput(makeContext(), nil, middleware2.AuthSubject{}, service.ContentModerationProtocolOpenAIResponses, "", nil)
	require.NotEmpty(t, first.AuditEventID)
	require.Equal(t, first.AuditEventID, same.AuditEventID)
	require.NotEqual(t, first.AuditEventID, another.AuditEventID)
	require.Equal(t, "content_moderation_unavailable", contentModerationErrorCode(&service.ContentModerationDecision{ErrorCode: "content_moderation_unavailable"}))
	require.Equal(t, "content_policy_violation", contentModerationErrorCode(&service.ContentModerationDecision{Flagged: true}))
}
