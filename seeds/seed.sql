-- Idempotent seed: insert only when the table is empty
INSERT INTO todos (title, completed, due_date, tags, created_at, updated_at)
SELECT title, completed, due_date, tags, NOW(), NOW()
FROM (VALUES
  ('Buy groceries',    false, '2026-04-01T12:00:00Z'::timestamptz, '{shopping,personal}'::text[]),
  ('Write unit tests', true,  '2026-03-15T18:00:00Z'::timestamptz, '{dev,testing}'::text[]),
  ('Deploy to k8s',    false, '2026-04-10T09:00:00Z'::timestamptz, '{devops,infra}'::text[])
) AS seed(title, completed, due_date, tags)
WHERE NOT EXISTS (SELECT 1 FROM todos LIMIT 1);