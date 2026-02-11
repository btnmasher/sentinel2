export type SearchResult = {
  character_record_id: string;
  character_id: number;
  name: string;
  user_id: string;
  is_main: boolean;
  main_name: string;
};

export type Character = {
  id: string;
  character_id: number;
  name: string;
  corp_id: number;
  corp_name?: string;
  alliance_id: number;
  alliance_name?: string;
  is_main: boolean;
  scopes: string;
  esi_last_refresh_at: string;
  esi_last_error: string;
  esi_token_valid: boolean;
};

export type UserDetails = {
  user_id: string;
  access_level: string;
  session_revoked_at: string;
  uploader_token_valid?: boolean;
  characters: Character[];
};

export type AuditEntry = {
  id: string;
  action: string;
  summary: string;
  actor_id: string;
  actor_display_name: string;
  target_user_id: string;
  target_user_name: string;
  target_character_id: number;
  target_character_name: string;
  created: string;
};

export type JobRun = {
  id: string;
  job_id: string;
  kind: string;
  step?: string;
  trigger?: string;
  actor_id?: string;
  actor_display_name?: string;
  status?: string;
  started_at?: string;
  completed_at?: string;
  duration_ms?: number;
  error?: string;
};

export type JobRunGroup = {
  parent: JobRun;
  steps: JobRun[];
};
