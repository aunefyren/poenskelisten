# WIP

Future and in-progress ideas only. Once something here is finished, delete its entry or move the relevant content into `README.md`/`docs/development.md`/`CLAUDE.md` — don't leave completed items in this file.

## Known bugs to fix

Found while writing controller tests to raise coverage; not yet fixed.

- **`RemoveSelfFromGroup` (`controllers/group.go`) is broken for everyone.** It builds an unbound `models.GroupMembership{}` and checks ownership using its zero-value `MemberID` (`uuid.Nil`) instead of the caller's own ID. GORM silently drops that zero-value field from the `Where` struct condition, so the ownership check degrades to "does an enabled group with this ID exist" — true for basically any group. The handler then always takes its "owners cannot remove themselves" branch, so **no one can currently leave a group** through this endpoint. Likely fix: check ownership against the caller's real `userID`.
- **`OIDCCallback` (`controllers/oidc.go`) never actually clears its state/nonce cookies.** The cleanup is `defer`red after `ctx.Redirect(...)`, but gin finalizes response headers at redirect time, so the deferred `SetCookie` calls run too late on every path (success or failure).
- **`DeleteWish` (`controllers/wish.go`)** returns `500` instead of `404`/`400` when the wish doesn't exist.
- **`GetWishlists`'s `?top=` filter (`controllers/wishlist.go`)** has an off-by-one and fails to truncate when the result count is exactly `top+1`.
- **`GetWishlists`'s `?group=` filter (`controllers/wishlist.go`)** returns `500` for a caller who isn't a member of the group, where the equivalent check elsewhere (`GetWishlist`, `DeleteWishlist`, ...) returns `400`.
