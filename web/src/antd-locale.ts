import zhCN from 'ant-design-vue/es/locale/zh_CN'
import type { Locale } from 'ant-design-vue/es/locale'

/**
 * ant-design-vue 自带 zh_CN 缺少 Table.emptyText，表格无数据时会显示英文「No data」。
 * 在此合并完整中文文案。
 */
const antdZhCN: Locale = {
  ...zhCN,
  Table: {
    ...zhCN.Table,
    emptyText: '暂无数据',
  },
}

export default antdZhCN
