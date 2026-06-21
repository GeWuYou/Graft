type ContainerOverviewTagTheme = 'default' | 'primary' | 'success' | 'warning' | 'danger';

/**
 * 容器详情概览行的页面视图模型。
 *
 * 这里显式保留 copy、text、tag、ports 四种联合分支，是为了把详情页
 * 的展示形态约束收口在 overview 边界内：上游页面只负责组装稳定字段，
 * 组件层根据 type 渲染交互、状态标签和端口列表，避免在模板里散落字段推断。
 */
export type ContainerOverviewInfoRow =
  | {
      code?: boolean;
      copyValue?: string;
      displayValue: string;
      key: string;
      label: string;
      testId?: string;
      type: 'copy';
    }
  | {
      displayValue: string;
      key: string;
      label: string;
      type: 'text';
    }
  | {
      key: string;
      label: string;
      tagLabel: string;
      tagTheme: ContainerOverviewTagTheme;
      type: 'tag';
    }
  | {
      emptyLabel: string;
      key: string;
      label: string;
      ports: string[];
      type: 'ports';
    };

/**
 * 容器详情概览分区。
 *
 * 分区只描述标题与行集合，不承载接口 DTO 或跨模块契约语义；后续新增
 * 概览字段时应优先扩展行联合类型，再让面板组件消费新的显式展示分支。
 */
export type ContainerOverviewInfoSection = {
  key: string;
  rows: ContainerOverviewInfoRow[];
  title: string;
};
