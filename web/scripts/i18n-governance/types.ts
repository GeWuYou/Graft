export type Severity = 'error' | 'warning';

export type SourceKind = 'vue' | 'ts' | 'tsx' | 'locale' | 'go' | 'schema';

export type SourceFile = {
  absolutePath: string;
  relativePath: string;
  kind: SourceKind;
  source: string;
  lineStarts: number[];
};

export type ScanContext = {
  rootDir: string;
  repositoryDir: string;
  srcDir: string;
  sourceFiles: SourceFile[];
  serverFiles: SourceFile[];
  strictKeyFirst: boolean;
};

export type RuleContext = ScanContext;

export interface I18nGovernanceRule {
  id: string;
  description: string;
  defaultSeverity: Severity;
  appliesTo: SourceKind[];
  check(context: RuleContext): RuleViolation[];
}

export interface RuleViolation {
  ruleId: string;
  severity: Severity;
  filePath: string;
  line: number;
  column?: number;
  message: string;
  excerpt?: string;
  suggestion?: string;
}
