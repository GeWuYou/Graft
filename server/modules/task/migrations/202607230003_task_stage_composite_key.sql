-- atlas:txmode none

-- 复合外键需要任务与阶段主键组合具备可引用的唯一性；并发建索引避免长时间阻塞阶段读写。
CREATE UNIQUE INDEX CONCURRENTLY task_stages_task_id_id_key_idx ON task_stages (task_id, id);

ALTER TABLE task_stages ADD CONSTRAINT task_stages_task_id_id_key UNIQUE USING INDEX task_stages_task_id_id_key_idx;
