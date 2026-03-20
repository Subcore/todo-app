INSERT INTO todos (title, completed, due_date, tags, created_at, updated_at)
VALUES
  ('Buy groceries', false, '2026-04-01T12:00:00Z',
   '{"shopping","personal"}', NOW(), NOW()),
  ('Write unit tests', true, '2026-03-15T18:00:00Z',
   '{"dev","testing"}', NOW(), NOW()),
  ('Deploy to k8s', false, '2026-04-10T09:00:00Z',
   '{"devops","infra"}', NOW(), NOW())
ON CONFLICT DO NOTHING;
