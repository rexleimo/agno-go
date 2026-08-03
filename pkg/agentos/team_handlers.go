package agentos

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rexleimo/agno-go/pkg/hno/models"
)

// TeamToolsResponse is the API payload for listing team tools.
type TeamToolsResponse struct {
	TeamID string                  `json:"team_id"`
	Tools  []models.ToolDefinition `json:"tools"`
}

// handleTeamTools returns aggregated tool definitions for a given team.
// GET /api/v1/teams/:id/tools
func (s *Server) handleTeamTools(c *gin.Context) {
	teamID := c.Param("id")
	if teamID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Status: "error",
			Error:  "team ID is required",
			Code:   "INVALID_REQUEST",
		})
		return
	}

	reg := s.teamRegistry
	if reg == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Status:  "error",
			Error:   "team not found",
			Message: "no teams registered",
			Code:    "TEAM_NOT_FOUND",
		})
		return
	}

	tm, err := reg.Get(teamID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Status:  "error",
			Error:   "team not found",
			Message: err.Error(),
			Code:    "TEAM_NOT_FOUND",
		})
		return
	}

	defs := TeamToolDefinitions(tm)
	resp := TeamToolsResponse{
		TeamID: teamID,
		Tools:  defs,
	}
	c.JSON(http.StatusOK, resp)
}
