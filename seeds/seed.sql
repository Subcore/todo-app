-- Idempotent seed: skip when the table already has data,
-- unless :force is set to 'on' (e.g. psql -v force=on / `make seed FORCE=1`).
\if :{?force}
\else
  \set force off
\endif

SELECT set_config('seed.force', :'force', false);

DO $$
BEGIN
  IF current_setting('seed.force') <> 'on' AND EXISTS (SELECT 1 FROM todos LIMIT 1) THEN
    RAISE NOTICE 'Seed skipped: todos table already has data. Re-run with FORCE=1 to insert anyway.';
  ELSE
    INSERT INTO todos (title, completed, due_date, tags, created_at, updated_at)
    SELECT title, completed, due_date, tags, NOW(), NOW()
    FROM (VALUES
      ('Buy groceries',    false, '2026-04-01T12:00:00Z'::timestamptz, '{shopping,personal}'::text[]),
      ('Write unit tests', true,  '2026-03-15T18:00:00Z'::timestamptz, '{dev,testing}'::text[]),
      ('Deploy to k8s',    false, '2026-04-10T09:00:00Z'::timestamptz, '{devops,infra}'::text[])
    ) AS seed(title, completed, due_date, tags);
    RAISE NOTICE 'Seed complete: 3 rows inserted.';
  END IF;
END
$$;
