#!/bin/bash
# 数据库迁移脚本
# 使用 golang-migrate 工具执行迁移

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 默认配置
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-postgres}
DB_PASSWORD=${DB_PASSWORD:-postgres}
DB_NAME=${DB_NAME:-smart_locker}
DB_SSLMODE=${DB_SSLMODE:-disable}
MIGRATIONS_DIR=${MIGRATIONS_DIR:-./migrations}

# 数据库 URL
DATABASE_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE}"

# 检查 migrate 工具是否安装
check_migrate() {
    if ! command -v migrate &> /dev/null; then
        echo -e "${YELLOW}migrate tool not found. Installing...${NC}"
        go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
        echo -e "${GREEN}migrate installed successfully${NC}"
    fi
}

# 显示帮助信息
show_help() {
    echo "Usage: $0 <command> [options]"
    echo ""
    echo "Commands:"
    echo "  up              Run all pending migrations"
    echo "  down            Rollback the last migration"
    echo "  down-all        Rollback all migrations"
    echo "  reset           Rollback all migrations and run them again"
    echo "  status          Show migration status"
    echo "  version         Show current migration version"
    echo "  create <name>   Create a new migration file"
    echo "  goto <version>  Migrate to a specific version"
    echo "  force <version> Force set migration version (use with caution)"
    echo ""
    echo "Environment variables:"
    echo "  DB_HOST         Database host (default: localhost)"
    echo "  DB_PORT         Database port (default: 5432)"
    echo "  DB_USER         Database user (default: postgres)"
    echo "  DB_PASSWORD     Database password (default: postgres)"
    echo "  DB_NAME         Database name (default: smart_locker)"
    echo "  DB_SSLMODE      SSL mode (default: disable)"
    echo "  MIGRATIONS_DIR  Migrations directory (default: ./migrations)"
}

# 运行迁移
run_up() {
    echo -e "${GREEN}Running migrations...${NC}"
    migrate -path "${MIGRATIONS_DIR}" -database "${DATABASE_URL}" up
    echo -e "${GREEN}Migrations completed successfully${NC}"
}

# 回滚迁移
run_down() {
    echo -e "${YELLOW}Rolling back last migration...${NC}"
    migrate -path "${MIGRATIONS_DIR}" -database "${DATABASE_URL}" down 1
    echo -e "${GREEN}Rollback completed successfully${NC}"
}

# 回滚所有迁移
run_down_all() {
    echo -e "${RED}Warning: This will rollback ALL migrations!${NC}"
    read -p "Are you sure? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        migrate -path "${MIGRATIONS_DIR}" -database "${DATABASE_URL}" down -all
        echo -e "${GREEN}All migrations rolled back${NC}"
    else
        echo -e "${YELLOW}Operation cancelled${NC}"
    fi
}

# 重置迁移
run_reset() {
    echo -e "${RED}Warning: This will rollback ALL migrations and run them again!${NC}"
    read -p "Are you sure? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        migrate -path "${MIGRATIONS_DIR}" -database "${DATABASE_URL}" down -all
        migrate -path "${MIGRATIONS_DIR}" -database "${DATABASE_URL}" up
        echo -e "${GREEN}Database reset completed${NC}"
    else
        echo -e "${YELLOW}Operation cancelled${NC}"
    fi
}

# 显示迁移状态
show_status() {
    echo -e "${GREEN}Migration status:${NC}"
    migrate -path "${MIGRATIONS_DIR}" -database "${DATABASE_URL}" version
}

# 显示当前版本
show_version() {
    migrate -path "${MIGRATIONS_DIR}" -database "${DATABASE_URL}" version
}

# 创建新迁移
create_migration() {
    if [ -z "$1" ]; then
        echo -e "${RED}Error: Migration name required${NC}"
        echo "Usage: $0 create <name>"
        exit 1
    fi

    # 获取下一个迁移编号
    LAST_NUM=$(ls -1 "${MIGRATIONS_DIR}"/*.sql 2>/dev/null | sed 's/.*\///' | sed 's/_.*//' | sort -n | tail -1)
    if [ -z "$LAST_NUM" ]; then
        NEXT_NUM="000001"
    else
        NEXT_NUM=$(printf "%06d" $((10#$LAST_NUM + 1)))
    fi

    UP_FILE="${MIGRATIONS_DIR}/${NEXT_NUM}_$1.up.sql"
    DOWN_FILE="${MIGRATIONS_DIR}/${NEXT_NUM}_$1.down.sql"

    echo "-- ${NEXT_NUM}_$1.up.sql" > "$UP_FILE"
    echo "-- Migration: $1" >> "$UP_FILE"
    echo "" >> "$UP_FILE"

    echo "-- ${NEXT_NUM}_$1.down.sql" > "$DOWN_FILE"
    echo "-- Rollback: $1" >> "$DOWN_FILE"
    echo "" >> "$DOWN_FILE"

    echo -e "${GREEN}Created migration files:${NC}"
    echo "  $UP_FILE"
    echo "  $DOWN_FILE"
}

# 迁移到指定版本
goto_version() {
    if [ -z "$1" ]; then
        echo -e "${RED}Error: Version number required${NC}"
        echo "Usage: $0 goto <version>"
        exit 1
    fi

    echo -e "${YELLOW}Migrating to version $1...${NC}"
    migrate -path "${MIGRATIONS_DIR}" -database "${DATABASE_URL}" goto "$1"
    echo -e "${GREEN}Migration completed${NC}"
}

# 强制设置版本
force_version() {
    if [ -z "$1" ]; then
        echo -e "${RED}Error: Version number required${NC}"
        echo "Usage: $0 force <version>"
        exit 1
    fi

    echo -e "${RED}Warning: This will force set the migration version to $1!${NC}"
    read -p "Are you sure? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        migrate -path "${MIGRATIONS_DIR}" -database "${DATABASE_URL}" force "$1"
        echo -e "${GREEN}Version forced to $1${NC}"
    else
        echo -e "${YELLOW}Operation cancelled${NC}"
    fi
}

# 主程序
main() {
    check_migrate

    case "$1" in
        up)
            run_up
            ;;
        down)
            run_down
            ;;
        down-all)
            run_down_all
            ;;
        reset)
            run_reset
            ;;
        status)
            show_status
            ;;
        version)
            show_version
            ;;
        create)
            create_migration "$2"
            ;;
        goto)
            goto_version "$2"
            ;;
        force)
            force_version "$2"
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            echo -e "${RED}Unknown command: $1${NC}"
            show_help
            exit 1
            ;;
    esac
}

main "$@"
