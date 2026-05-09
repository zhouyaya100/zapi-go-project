package core

import (
	"github.com/zapi/zapi-go/internal/model"
)

// InvalidateUserAndDependencies invalidates user cache plus all related caches
// Call this whenever a user is updated
func InvalidateUserAndDependencies(user *model.User) {
	InvalidateUserCache(user.ID)
	// If user's group changed, invalidate old and new group caches
	if user.GroupID != nil {
		InvalidateGroupCache(*user.GroupID)
	}
	// Invalidate all token caches for this user (quota may have changed)
	InvalidateAllTokenCache()
}

// InvalidateGroupAndDependencies invalidates group cache plus all user caches in that group
// Call this whenever a group is updated
func InvalidateGroupAndDependencies(groupID uint) {
	InvalidateGroupCache(groupID)
	// All users in this group need fresh data
	InvalidateAllUserCache()
	InvalidateAllTokenCache()
}

// InvalidateChannelAndDependencies invalidates channel cache plus routing pool
// Call this whenever a channel is updated
func InvalidateChannelAndDependencies(channelID uint) {
	InvalidateAllTokenCache()
	InvalidateAllUserCache()
}

// SafeCreateUser creates a user and invalidates caches
func SafeCreateUser(user *model.User) error {
	if err := model.DB.Create(user).Error; err != nil {
		return err
	}
	InvalidateUserAndDependencies(user)
	return nil
}

// SafeUpdateUser updates a user and invalidates caches
func SafeUpdateUser(user *model.User) error {
	if err := model.DB.Save(user).Error; err != nil {
		return err
	}
	InvalidateUserAndDependencies(user)
	return nil
}

// SafeCreateGroup creates a group and invalidates caches
func SafeCreateGroup(group *model.Group) error {
	if err := model.DB.Create(group).Error; err != nil {
		return err
	}
	InvalidateGroupAndDependencies(group.ID)
	return nil
}

// SafeUpdateGroup updates a group and invalidates caches
func SafeUpdateGroup(group *model.Group) error {
	if err := model.DB.Save(group).Error; err != nil {
		return err
	}
	InvalidateGroupAndDependencies(group.ID)
	return nil
}
