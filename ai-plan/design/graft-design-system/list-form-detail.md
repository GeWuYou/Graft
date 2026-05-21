# List / Form / Detail Template

## Intent

Use this for CRUD modules and most business capabilities.

## Structure

1. Page header
2. Primary action area or filter area
3. Main content surface
4. Row actions
5. Drawer or dialog form
6. Detail panel, empty state, or feedback surface

## Notes

- Prefer table-first layouts for dense data.
- Keep operation columns explicit.
- Use drawers for side editing and dialogs for lightweight tasks.

## Do

- Use `Card`, `Form`, `Table`, `Drawer`, `Dialog`, `Tag`, `Pagination`.

## Don’t

- Split the same data into too many competing panels.
- Hide operations in custom click-only controls.
- Recreate starter demo pages as new truth.
