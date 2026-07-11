package models

import "github.com/google/uuid"

type Wish struct {
	GormModel
	Name       string     `json:"name" gorm:"not null"`
	Note       string     `json:"note"`
	Price      *float64   `json:"price" gorm:"default: null"`
	Enabled    bool       `json:"enabled" gorm:"not null; default: true"`
	OwnerID    uuid.UUID  `json:"" gorm:"type:varchar(100);"`
	Owner      User       `json:"owner" gorm:"not null;"`
	URL        string     `json:"url"`
	WishlistID uuid.UUID  `json:"" gorm:"type:varchar(100);"`
	Wishlist   Wishlist   `json:"wishlist" gorm:"not null;"`
	CategoryID *uuid.UUID `json:"" gorm:"type:varchar(100);default:null"`
}

type WishCreationRequest struct {
	Name         string     `json:"name"`
	Note         string     `json:"note"`
	Price        *float64   `json:"price"`
	URL          string     `json:"url"`
	Image        string     `json:"image_data"`
	CategoryID   *uuid.UUID `json:"category_id"`
	CategoryName string     `json:"category_name"`
}

type WishUpdateRequest struct {
	Name         string     `json:"name"`
	Note         string     `json:"note"`
	Price        *float64   `json:"price"`
	URL          string     `json:"url"`
	Image        string     `json:"image_data"`
	ImageDelete  bool       `json:"image_delete"`
	CategoryID   *uuid.UUID `json:"category_id"`
	CategoryName string     `json:"category_name"`
}

type WishObject struct {
	GormModel
	Name            string                       `json:"name"`
	Note            string                       `json:"note"`
	Price           *float64                     `json:"price"`
	Enabled         bool                         `json:"enabled"`
	Owner           User                         `json:"owner_id"`
	WishlistOwner   User                         `json:"wishlist_owner"`
	Collaborators   []WishlistCollaboratorObject `json:"collaborators"`
	URL             string                       `json:"url"`
	Image           bool                         `json:"image"`
	WishlistID      uuid.UUID                    `json:"wishlist_id"`
	WishClaim       []WishClaimObject            `json:"wishclaim"`
	WishClaimable   bool                         `json:"wish_claimable"`
	Currency        string                       `json:"currency"`
	CurrencyPadding bool                         `json:"currency_padding"`
	CurrencyLeft    bool                         `json:"currency_left"`
	Category        *WishCategoryObject          `json:"category"`
}

// WishCategory groups similar wishes within a single wishlist so they can be
// collapsed/expanded together. A wish belongs to zero or one category via
// Wish.CategoryID. It is intentionally distinct from models.Group, which
// governs wishlist sharing, not wish grouping.
type WishCategory struct {
	GormModel
	Name       string    `json:"name" gorm:"not null"`
	WishlistID uuid.UUID `json:"" gorm:"type:varchar(100);"`
	OwnerID    uuid.UUID `json:"" gorm:"type:varchar(100);"`
	SortOrder  int       `json:"sort_order" gorm:"not null;default:0"`
	Enabled    bool      `json:"enabled" gorm:"not null;default:true"`
}

type WishCategoryObject struct {
	GormModel
	Name       string    `json:"name"`
	WishlistID uuid.UUID `json:"wishlist_id"`
	SortOrder  int       `json:"sort_order"`
}

type WishClaim struct {
	GormModel
	WishID  uuid.UUID `json:"" gorm:"type:varchar(100);"`
	Wish    Wish      `json:"wish" gorm:"not null;"`
	UserID  uuid.UUID `json:"" gorm:"type:varchar(100);"`
	User    User      `json:"user" gorm:"not null;"`
	Enabled bool      `json:"enabled" gorm:"not null;default: true"`
}

type WishClaimObject struct {
	GormModel
	Wish    uuid.UUID   `json:"wish_id"`
	User    UserMinimal `json:"user"`
	Enabled bool        `json:"enabled"`
}

type WishClaimCreationRequest struct {
	WishlistID *uuid.UUID `json:"wishlist_id"`
}
