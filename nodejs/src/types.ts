/**
 * Axon Node.js SDK - Type definitions
 */

/**
 * Session information
 */
export interface SessionInfo {
  session_id: string;
  status: string;
  profile?: string;
  created_at?: string;
  last_action?: string;
  url?: string;
  title?: string;
  auth_state?: string;
  page_state?: string;
}

/**
 * Request to create a session
 */
export interface CreateSessionRequest {
  id: string;
  profile?: string;
}

/**
 * Response from creating a session
 */
export interface CreateSessionResponse {
  session_id: string;
  status: string;
  profile?: string;
}

/**
 * An element in the accessibility tree with spatial metadata
 */
export interface SnapshotElement {
  ref: string;
  type: string;
  label: string;
  role?: string;
  value?: string;
  x: number;
  y: number;
  width: number;
  height: number;
  visible: boolean;
  enabled: boolean;
  intent?: string;
  reversible?: string;
  related_ref?: string;
  vault_suggestion?: string;
}

/**
 * Response from taking a snapshot
 */
export interface SnapshotResponse {
  session_id: string;
  url: string;
  title: string;
  elements: SnapshotElement[];
  page_state?: string;
  captcha_detected: boolean;
  timestamp?: string;
  token_count: number;
  content: string;
}

/**
 * Request to perform an action
 */
export interface ActionRequest {
  action: string;
  ref: string;
  value?: string;
  confirm?: boolean;
}

/**
 * Response from performing an action
 */
export interface ActionResponse {
  success: boolean;
  session_id: string;
  action: string;
  message?: string;
  error?: string;
  result?: string;
  requires_confirm?: boolean;
}

/**
 * Request to navigate to a URL
 */
export interface NavigateRequest {
  url: string;
  wait_until?: 'none' | 'load' | 'domcontentloaded' | 'networkidle';
}

/**
 * Response from navigating
 */
export interface NavigateResponse {
  session_id: string;
  url: string;
  success: boolean;
  title?: string;
  state?: string;
}

/**
 * A single frame in a session replay
 */
export interface ReplayFrame {
  timestamp: string;
  data: string; // Base64 encoded image
  url: string;
  metadata?: Record<string, unknown>;
}

/**
 * Response from getting a session replay
 */
export interface ReplayResponse {
  session_id: string;
  frames: ReplayFrame[];
}

/**
 * API error response
 */
export interface APIError {
  error: boolean;
  error_type: string;
  message: string;
  recoverable: boolean;
}

/**
 * List of sessions
 */
export interface SessionList {
  sessions: SessionInfo[];
}

/**
 * Find and act request
 */
export interface FindAndActRequest {
  action: string;
  intent: string;
  value?: string;
}

/**
 * Action type
 */
export type ActionType = 'click' | 'fill' | 'hover' | 'select' | 'check' | 'uncheck';

/**
 * Axon client configuration
 */
export interface AxonOptions {
  apiUrl?: string;
  timeout?: number;
}
