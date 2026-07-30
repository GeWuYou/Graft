DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'update_operations'
          AND column_name = 'update_mode'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'update_operations'
          AND column_name = 'deployment_strategy'
    ) THEN
        ALTER TABLE update_operations
            RENAME COLUMN update_mode TO deployment_strategy;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = 'update_operations'::regclass
          AND conname = 'update_operations_update_mode_check'
    ) AND NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = 'update_operations'::regclass
          AND conname = 'update_operations_deployment_strategy_check'
    ) THEN
        ALTER TABLE update_operations
            RENAME CONSTRAINT update_operations_update_mode_check
            TO update_operations_deployment_strategy_check;
    END IF;
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'update_operations'
          AND column_name = 'deployment_strategy'
    ) THEN
        COMMENT ON COLUMN update_operations.deployment_strategy IS '创建更新操作时冻结的部署升级策略；历史记录无法推导时标记为未知';
    END IF;
END $$;
