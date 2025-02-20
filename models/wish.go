package models

import "github.com/google/uuid"

type Wish struct {
	GormModel
	Name       string    `json:"name" gorm:"not null"`
	Note       string    `json:"note"`
	Price      *float64  `json:"price" gorm:"default: null"`
	Enabled    bool      `json:"enabled" gorm:"not null; default: true"`
	OwnerID    uuid.UUID `json:"" gorm:"type:varchar(100);"`
	Owner      User      `json:"owner" gorm:"not null;"`
	URL        string    `json:"url"`
	WishlistID uuid.UUID `json:"" gorm:"type:varchar(100);"`
	Wishlist   Wishlist  `json:"wishlist" gorm:"not null;"`
}

type WishCreationRequest struct {
	Name  string   `json:"name"`
	Note  string   `json:"note"`
	Price *float64 `json:"price"`
	URL   string   `json:"url"`
	Image string   `json:"image_data"`
}

type WishUpdateRequest struct {
	Name  string   `json:"name"`
	Note  string   `json:"note"`
	Price *float64 `json:"price"`
	URL   string   `json:"url"`
	Image string   `json:"image_data"`
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
