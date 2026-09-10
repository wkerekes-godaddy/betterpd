package pd

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/PagerDuty/go-pagerduty"
)

type Client struct {
	api     *pagerduty.Client
	userID  string
	email   string
	teamIDs []string
	orgName string
}

func NewClient(token string, userID string, userEmail string, opts ...pagerduty.ClientOptions) (*Client, error) {
	api := pagerduty.NewClient(token, opts...)
	c := &Client{api: api, email: userEmail}

	if userID == "" {
		return nil, fmt.Errorf("no user_id configured. Run 'betterpd --config' to set up")
	}
	c.userID = userID

	// Fetch user details to get team memberships
	ctx := context.Background()
	user, err := api.GetUserWithContext(ctx, userID, pagerduty.GetUserOptions{})
	if err != nil {
		return nil, fmt.Errorf("fetching user %s: %w", userID, err)
	}

	// Always trust the email PagerDuty has on file for this user_id over
	// any manually-configured value: PagerDuty rejects the "From" header
	// on incident actions with "Requester User Not Found" (1008) if it
	// doesn't exactly match a real user's registered email.
	if user.Email != "" {
		c.email = user.Email
	}
	for _, team := range user.Teams {
		c.teamIDs = append(c.teamIDs, team.ID)
	}
	c.orgName = subdomainFromURL(user.HTMLURL)

	return c, nil
}

// OrgName returns the PagerDuty account subdomain (e.g. "acme" from
// acme.pagerduty.com), derived from the current user's HTML URL.
func (c *Client) OrgName() string {
	return c.orgName
}

func subdomainFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	host := strings.TrimSuffix(u.Hostname(), ".pagerduty.com")
	if host == u.Hostname() {
		return ""
	}
	return host
}

func (c *Client) ListIncidents(ctx context.Context, scope Scope) ([]Incident, error) {
	opts := pagerduty.ListIncidentsOptions{
		Statuses: []string{"triggered", "acknowledged"},
		SortBy:   "created_at:desc",
	}

	switch scope {
	case ScopeMine:
		opts.UserIDs = []string{c.userID}
	case ScopeTeam:
		opts.TeamIDs = c.teamIDs
	case ScopeAll:
		// no filter
	}

	var all []Incident
	opts.Offset = 0
	opts.Limit = 100

	for {
		resp, err := c.api.ListIncidentsWithContext(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("listing incidents: %w", err)
		}

		for _, inc := range resp.Incidents {
			all = append(all, convertIncident(inc))
		}

		if !resp.More {
			break
		}
		opts.Offset += opts.Limit
	}

	return all, nil
}

func (c *Client) Acknowledge(ctx context.Context, ids []string) error {
	return c.manageIncidents(ctx, ids, "acknowledged")
}

func (c *Client) Resolve(ctx context.Context, ids []string) error {
	return c.manageIncidents(ctx, ids, "resolved")
}

func (c *Client) Snooze(ctx context.Context, id string, duration time.Duration) error {
	_, err := c.api.SnoozeIncidentWithContext(ctx, id, uint(duration.Seconds()))
	if err != nil {
		return fmt.Errorf("snoozing incident: %w", err)
	}
	return nil
}

func (c *Client) manageIncidents(ctx context.Context, ids []string, status string) error {
	var incidents []pagerduty.ManageIncidentsOptions
	for _, id := range ids {
		incidents = append(incidents, pagerduty.ManageIncidentsOptions{
			ID:     id,
			Status: status,
			Type:   "incident_reference",
		})
	}

	_, err := c.api.ManageIncidentsWithContext(ctx, c.email, incidents)
	if err != nil {
		return fmt.Errorf("managing incidents (%s): %w", status, err)
	}
	return nil
}

func convertIncident(inc pagerduty.Incident) Incident {
	i := Incident{
		ID:      inc.ID,
		Number:  inc.IncidentNumber,
		Title:   inc.Title,
		Status:  inc.Status,
		Urgency: inc.Urgency,
		HTMLURL: inc.HTMLURL,
	}

	if inc.Service.Summary != "" {
		i.ServiceName = inc.Service.Summary
	}

	if len(inc.Assignments) > 0 {
		i.AssignedTo = inc.Assignments[0].Assignee.Summary
	}

	if t, err := time.Parse(time.RFC3339, inc.CreatedAt); err == nil {
		i.CreatedAt = t
	}

	// Note: SnoozedUntil is deliberately NOT derived from inc.PendingActions
	// here. PagerDuty represents a snooze as an acknowledge plus a scheduled
	// "unacknowledge" pending action, but many accounts also auto-schedule an
	// "unacknowledge" pending action on every plain acknowledge (an ack
	// timeout policy) — so that signal can't distinguish a real snooze from
	// an ordinary ack. Snooze state is tracked client-side instead; see App.

	return i
}
