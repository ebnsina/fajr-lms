-- name: CreatePlan :one
INSERT INTO payment_plans (tenant_id, user_id, course_id, total_minor, currency, parts, gap_days, next_due_on)
VALUES (@tenant_id, @user_id, @course_id, @total_minor, @currency, @parts, @gap_days, current_date)
RETURNING *;

-- name: GetPlan :one
SELECT * FROM payment_plans WHERE id = @id;

-- name: PlanForCourse :one
SELECT * FROM payment_plans WHERE course_id = @course_id AND user_id = @user_id;

-- name: MyPlans :many
SELECT sqlc.embed(p), c.title, c.slug
FROM payment_plans p JOIN courses c ON c.id = p.course_id
WHERE p.user_id = @user_id
ORDER BY p.next_due_on NULLS LAST;

-- name: ListPlans :many
SELECT sqlc.embed(p), c.title, u.full_name
FROM payment_plans p JOIN courses c ON c.id = p.course_id JOIN users u ON u.id = p.user_id
ORDER BY p.next_due_on NULLS LAST
LIMIT @page_limit OFFSET @page_offset;

-- name: CountPlanPart :one
-- Counts a part as paid and moves the due date on. The whole plan being paid
-- off leaves no next date.
UPDATE payment_plans SET
  paid_parts = paid_parts + 1,
  next_due_on = CASE WHEN paid_parts + 1 >= parts THEN NULL
                     ELSE current_date + (gap_days || ' days')::interval END
WHERE id = @id AND paid_parts < parts
RETURNING *;
