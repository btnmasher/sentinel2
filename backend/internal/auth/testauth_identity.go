package auth

import (
	"context"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"sentinel2/internal/store"
)

// SelectMainCharacter returns the current primary character from TestAuth user info.
func SelectMainCharacter(userInfo *UserInfo) (CharacterInfo, bool) {
	if userInfo == nil || len(userInfo.Characters) == 0 {
		return CharacterInfo{}, false
	}

	if userInfo.MainCharacterID != nil {
		for _, character := range userInfo.Characters {
			if character.CharacterID == *userInfo.MainCharacterID {
				return character, true
			}
		}
	}

	for _, character := range userInfo.Characters {
		if character.IsPrimary {
			return character, true
		}
	}

	return CharacterInfo{}, false
}

func (p *TestAuthProvider) resolveAccessLevel(userInfo *UserInfo) string {
	if userInfo == nil {
		return accessLevelUser
	}

	if userInfo.IsAdmin {
		return accessLevelAdmin
	}

	if p.matchesAccessGrant(userInfo, p.Config.GetTestAuthAdminGroups(), p.Config.GetTestAuthAdminPermissionURNs()) {
		return accessLevelAdmin
	}

	if p.matchesAccessGrant(userInfo, p.Config.GetTestAuthStaffGroups(), p.Config.GetStaffPermissionURNs()) {
		return accessLevelStaff
	}

	return accessLevelUser
}

func (p *TestAuthProvider) matchesAccessGrant(userInfo *UserInfo, groups, permissionURNs []string) bool {
	if userInfo == nil {
		return false
	}

	targets := normalizeAccessGrantTargets(groups, permissionURNs)
	if len(targets) == 0 {
		return false
	}

	return userInfoHasAccessGrant(userInfo, targets)
}

func (p *TestAuthProvider) resolveCharacterAffiliation(ctx context.Context, userInfo *UserInfo) (corpID, allianceID int, err error) {
	if p == nil || p.PublicESI == nil || userInfo == nil {
		return 0, 0, ErrAccessDenied
	}

	mainCharacter, ok := SelectMainCharacter(userInfo)
	if !ok {
		return 0, 0, ErrAccessDenied
	}

	return p.PublicESI.CharacterAffiliation(ctx, int(mainCharacter.CharacterID))
}

func (p *TestAuthProvider) linkableCharacters(ctx context.Context, user *core.Record) ([]CharacterInfo, map[int]CharacterInfo, error) {
	userInfo, infoErr := p.currentUserInfo(ctx, user)
	if infoErr != nil {
		return nil, nil, infoErr
	}

	linkedIDs, linkedErr := p.linkedCharacterIDs(user)
	if linkedErr != nil {
		return nil, nil, linkedErr
	}

	mainCharacter, _ := SelectMainCharacter(userInfo)
	mainCharacterID := int64(0)
	if mainCharacter.CharacterID > 0 {
		mainCharacterID = mainCharacter.CharacterID
	}

	linkable := make([]CharacterInfo, 0, len(userInfo.Characters))
	linkableByID := make(map[int]CharacterInfo, len(userInfo.Characters))
	for _, character := range userInfo.Characters {
		if !character.HasValidToken {
			continue
		}
		if character.IsPrimary || character.CharacterID == mainCharacterID {
			continue
		}
		if _, exists := linkedIDs[int(character.CharacterID)]; exists {
			continue
		}
		linkable = append(linkable, character)
		linkableByID[int(character.CharacterID)] = character
	}

	return linkable, linkableByID, nil
}

func (p *TestAuthProvider) currentUserInfo(ctx context.Context, user *core.Record) (*UserInfo, error) {
	if p == nil || p.OAuth == nil || user == nil {
		return nil, ErrUnauthorized
	}

	accessToken := strings.TrimSpace(user.GetString("oauth_access_token"))
	if accessToken == "" {
		return nil, ErrMissingAccessToken
	}

	return p.OAuth.GetUserInfo(ctx, accessToken)
}

func (p *TestAuthProvider) linkedCharacterIDs(user *core.Record) (map[int]struct{}, error) {
	if p == nil || p.App == nil || user == nil {
		return map[int]struct{}{}, ErrUnauthorized
	}

	records, recordsErr := p.App.FindRecordsByFilter(
		store.CollectionCharacters,
		"user = {:user} && auth_provider = {:provider}",
		"",
		0,
		0,
		dbx.Params{
			"user":     user.Id,
			"provider": p.Name(),
		},
	)
	if recordsErr != nil {
		return nil, recordsErr
	}

	linkedIDs := make(map[int]struct{}, len(records))
	for _, record := range records {
		characterID := record.GetInt("eve_character_id")
		if characterID > 0 {
			linkedIDs[characterID] = struct{}{}
		}
	}
	return linkedIDs, nil
}

func (p *TestAuthProvider) authorizeAccess(corpID, allianceID int) error {
	allowed, accessErr := p.allowedAccess(corpID, allianceID)
	if accessErr != nil {
		return ErrFailedCheckAccess
	}
	if !allowed {
		return ErrAccessDenied
	}
	return nil
}

func (p *TestAuthProvider) allowedAccess(corpID, allianceID int) (bool, error) {
	corpAllowed, corpErr := p.allowedID("allowed_corporations", corpID)
	if corpErr != nil {
		return false, corpErr
	}
	allianceAllowed, allianceErr := p.allowedID("allowed_alliances", allianceID)
	if allianceErr != nil {
		return false, allianceErr
	}

	if corpAllowed || allianceAllowed {
		return true, nil
	}

	if !p.hasAllowlist("allowed_corporations") && !p.hasAllowlist("allowed_alliances") {
		return false, nil
	}
	return false, nil
}

func (p *TestAuthProvider) hasAllowlist(collection string) bool {
	records, recordsErr := p.App.FindRecordsByFilter(collection, "", "", 1, 0, nil)
	return recordsErr == nil && len(records) > 0
}

func (p *TestAuthProvider) allowedID(collection string, id int) (bool, error) {
	if id == 0 {
		return false, nil
	}
	records, recordsErr := p.App.FindRecordsByFilter(collection, "eve_id = {:id}", "", 1, 0, map[string]any{"id": id})
	if recordsErr != nil {
		return false, recordsErr
	}
	return len(records) > 0, nil
}

func normalizeAccessGrant(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeAccessGrantTargets(groups, permissionURNs []string) map[string]struct{} {
	targets := make(map[string]struct{}, len(groups)+len(permissionURNs))
	for _, entry := range groups {
		normalized := normalizeAccessGrant(entry)
		if normalized != "" {
			targets[normalized] = struct{}{}
		}
	}
	for _, entry := range permissionURNs {
		normalized := normalizeAccessGrant(entry)
		if normalized != "" {
			targets[normalized] = struct{}{}
		}
	}
	return targets
}

func userInfoHasAccessGrant(userInfo *UserInfo, targets map[string]struct{}) bool {
	if hasMatchingGrant(userInfo.Groups, targets) {
		return true
	}
	if hasMatchingMembership(userInfo.GroupMemberships, targets) {
		return true
	}
	return hasMatchingGrant(userInfo.PermissionURNs, targets)
}

func hasMatchingGrant(values []string, targets map[string]struct{}) bool {
	for _, value := range values {
		if _, ok := targets[normalizeAccessGrant(value)]; ok {
			return true
		}
	}
	return false
}

func hasMatchingMembership(memberships []GroupMembership, targets map[string]struct{}) bool {
	for _, membership := range memberships {
		if _, ok := targets[normalizeAccessGrant(membership.GroupID)]; ok {
			return true
		}
		if _, ok := targets[normalizeAccessGrant(membership.GroupName)]; ok {
			return true
		}
	}
	return false
}
