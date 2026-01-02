#!/bin/bash
# 种子数据加载脚本

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
SEEDS_DIR=${SEEDS_DIR:-./seeds}

# 导出密码环境变量
export PGPASSWORD="${DB_PASSWORD}"

# 检查 psql 是否可用
check_psql() {
    if ! command -v psql &> /dev/null; then
        echo -e "${RED}Error: psql command not found. Please install PostgreSQL client.${NC}"
        exit 1
    fi
}

# 执行 SQL 文件
execute_sql() {
    local file=$1
    echo -e "${YELLOW}Executing: $(basename "$file")${NC}"
    psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -f "$file"
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ $(basename "$file") executed successfully${NC}"
    else
        echo -e "${RED}✗ $(basename "$file") failed${NC}"
        exit 1
    fi
}

# 显示帮助
show_help() {
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  load    Load all seed data (default)"
    echo "  clean   Clear all data and reload seeds"
    echo "  help    Show this help message"
    echo ""
    echo "Environment variables:"
    echo "  DB_HOST      Database host (default: localhost)"
    echo "  DB_PORT      Database port (default: 5432)"
    echo "  DB_USER      Database user (default: postgres)"
    echo "  DB_PASSWORD  Database password (default: postgres)"
    echo "  DB_NAME      Database name (default: smart_locker)"
    echo "  SEEDS_DIR    Seeds directory (default: ./seeds)"
}

# 加载种子数据
load_seeds() {
    echo -e "${GREEN}Loading seed data...${NC}"
    echo ""

    # 按顺序执行种子文件
    local files=(
        "${SEEDS_DIR}/001_users.sql"
        "${SEEDS_DIR}/002_rbac.sql"
        "${SEEDS_DIR}/003_devices.sql"
        "${SEEDS_DIR}/004_hotels.sql"
        "${SEEDS_DIR}/005_products.sql"
        "${SEEDS_DIR}/006_marketing.sql"
        "${SEEDS_DIR}/007_system.sql"
    )

    for file in "${files[@]}"; do
        if [ -f "$file" ]; then
            execute_sql "$file"
        else
            echo -e "${YELLOW}Warning: $file not found, skipping${NC}"
        fi
    done

    echo ""
    echo -e "${GREEN}All seed data loaded successfully!${NC}"
}

# 清理并重新加载
clean_and_load() {
    echo -e "${RED}Warning: This will delete all existing data!${NC}"
    read -p "Are you sure? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}Cleaning data...${NC}"

        # 按依赖关系倒序删除数据
        psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" << 'EOF'
-- 禁用外键检查
SET session_replication_role = replica;

-- 清空表数据
TRUNCATE TABLE
    wallet_transactions,
    settlements,
    commissions,
    withdrawals,
    distributors,
    user_coupons,
    coupons,
    campaigns,
    member_packages,
    reviews,
    cart_items,
    product_skus,
    products,
    categories,
    bookings,
    room_time_slots,
    rooms,
    hotels,
    rentals,
    rental_pricings,
    refunds,
    payments,
    order_items,
    orders,
    device_maintenances,
    device_logs,
    devices,
    venues,
    merchants,
    operation_logs,
    sms_codes,
    notifications,
    articles,
    message_templates,
    system_configs,
    banners,
    addresses,
    user_feedbacks,
    user_wallets,
    admins,
    role_permissions,
    permissions,
    users
RESTART IDENTITY CASCADE;

-- 保留角色和会员等级
-- DELETE FROM roles WHERE is_system = FALSE;

-- 恢复外键检查
SET session_replication_role = DEFAULT;
EOF

        echo -e "${GREEN}Data cleaned successfully${NC}"
        echo ""
        load_seeds
    else
        echo -e "${YELLOW}Operation cancelled${NC}"
    fi
}

# 主程序
main() {
    check_psql

    case "${1:-load}" in
        load)
            load_seeds
            ;;
        clean)
            clean_and_load
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
