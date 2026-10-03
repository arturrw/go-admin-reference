-- name: ListProducts :many
SELECT * FROM products
WHERE (sqlc.arg(category)::text = '' OR category = sqlc.arg(category)::text)
  AND (sqlc.arg(status)::text = '' OR status = sqlc.arg(status)::text)
  AND (sqlc.arg(q)::text = ''
       OR (name || ' ' || sku || ' ' || vendor || ' ' || array_to_string(tags, ' ')) ILIKE '%' || sqlc.arg(q)::text || '%')
ORDER BY
  CASE WHEN sqlc.arg(sort)::text = 'sales' THEN sold_30d END DESC,
  CASE WHEN sqlc.arg(sort)::text = 'price' THEN price_cents END DESC,
  CASE WHEN sqlc.arg(sort)::text = 'stock' THEN stock END ASC,
  CASE WHEN sqlc.arg(sort)::text = 'name' THEN name END ASC,
  sold_30d::bigint * price_cents DESC,
  id;

-- name: ProductStats :one
SELECT count(*)::int                                            AS total,
       count(*) FILTER (WHERE status = 'active')::int           AS active,
       count(*) FILTER (WHERE stock > 0 AND stock < 15)::int    AS low_stock,
       count(*) FILTER (WHERE stock = 0)::int                   AS out_of_stock,
       coalesce(sum(stock::bigint * price_cents), 0)::bigint    AS inventory_value_cents
FROM products;

-- name: ProductCountsByCategory :many
SELECT category, count(*)::int AS n FROM products GROUP BY category;

-- name: ProductSalesByCategory :many
SELECT category, coalesce(sum(sold_30d::bigint * price_cents), 0)::bigint AS sales_cents
FROM products GROUP BY category;

-- name: TopProducts :many
SELECT * FROM products ORDER BY sold_30d::bigint * price_cents DESC, id LIMIT sqlc.arg(lim)::int;

-- name: GetProduct :one
SELECT * FROM products WHERE id = $1;

-- name: CreateProduct :one
INSERT INTO products (name, sku, category, vendor, tags, price_cents, compare_at_cents, cost_cents,
                      stock, weight_grams, status, description, hue, trend)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, '{0,0,0,0,0,0,0,0,0,0,0,0,0,0}')
RETURNING *;

-- name: UpdateProduct :one
UPDATE products
SET name             = sqlc.arg(name),
    sku              = sqlc.arg(sku),
    -- Right-hand side sees the old row, so the hue only changes with the category.
    hue              = CASE WHEN category <> sqlc.arg(category)::text THEN sqlc.arg(hue)::int ELSE hue END,
    category         = sqlc.arg(category)::text,
    vendor           = sqlc.arg(vendor),
    tags             = sqlc.arg(tags),
    price_cents      = sqlc.arg(price_cents),
    compare_at_cents = sqlc.arg(compare_at_cents),
    cost_cents       = sqlc.arg(cost_cents),
    stock            = sqlc.arg(stock),
    weight_grams     = sqlc.arg(weight_grams),
    status           = sqlc.arg(status),
    description      = sqlc.arg(description),
    updated_at       = now()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteProduct :execrows
DELETE FROM products WHERE id = $1;

-- name: SetProductsStatus :execrows
UPDATE products SET status = sqlc.arg(status), updated_at = now() WHERE id = ANY(sqlc.arg(ids)::bigint[]);

-- name: DeleteProducts :execrows
DELETE FROM products WHERE id = ANY(sqlc.arg(ids)::bigint[]);

-- name: TouchProduct :exec
UPDATE products SET updated_at = now() WHERE id = $1;

-- name: ListProductImages :many
SELECT * FROM product_images WHERE product_id = ANY(sqlc.arg(product_ids)::bigint[]) ORDER BY product_id, position, created_at;

-- name: CountProductImages :one
SELECT count(*)::int FROM product_images WHERE product_id = $1;

-- name: AddProductImage :exec
INSERT INTO product_images (id, product_id, position, url, alt, generated, size_bytes)
VALUES (sqlc.arg(id), sqlc.arg(product_id),
        (SELECT coalesce(max(pi.position) + 1, 0) FROM product_images pi WHERE pi.product_id = sqlc.arg(product_id)),
        sqlc.arg(url), sqlc.arg(alt), sqlc.arg(generated), sqlc.arg(size_bytes));

-- name: DeleteProductImage :one
DELETE FROM product_images WHERE product_id = $1 AND id = $2 RETURNING *;

-- name: SetPrimaryImage :execrows
-- Moves the image in front of the current first one.
UPDATE product_images AS t
SET position = m.min_position - 1
FROM (SELECT min(position) AS min_position FROM product_images WHERE product_id = sqlc.arg(product_id)) AS m
WHERE t.product_id = sqlc.arg(product_id) AND t.id = sqlc.arg(image_id);

-- name: ProductImageFiles :many
SELECT url FROM product_images WHERE product_id = ANY(sqlc.arg(product_ids)::bigint[]) AND NOT generated;
