import {
  agent as createAgentApp,
  methods,
  type AgentConnection,
  type AuthenticateRequest,
  type AuthenticateResponse,
  type CancelNotification,
  type CloseSessionRequest,
  type CloseSessionResponse,
  type DeleteSessionRequest,
  type DeleteSessionResponse,
  type InitializeRequest,
  type InitializeResponse,
  type ListSessionsRequest,
  type ListSessionsResponse,
  type LoadSessionRequest,
  type LoadSessionResponse,
  type NewSessionRequest,
  type NewSessionResponse,
  type PromptRequest,
  type PromptResponse,
  type RequestPermissionRequest,
  type RequestPermissionResponse,
  type ResumeSessionRequest,
  type ResumeSessionResponse,
  type SessionNotification,
  type SetSessionConfigOptionRequest,
  type SetSessionConfigOptionResponse,
  type SetSessionModeRequest,
  type SetSessionModeResponse,
  type Stream
} from '@agentclientprotocol/sdk'

/**
 * Adapter between the class-based agent in this package and the handler-based
 * `agent()` API of @agentclientprotocol/sdk 2.x.
 *
 * `PiAcpAgent` keeps its method-per-request shape. `serveAgent` registers one
 * handler per implemented method and exposes the two client calls the agent
 * uses (`sessionUpdate`, `requestPermission`) as `AgentSideConnection`.
 */

/** Client-side calls this adapter makes. */
export interface AgentSideConnection {
  sessionUpdate(params: SessionNotification): Promise<void>
  requestPermission(params: RequestPermissionRequest): Promise<RequestPermissionResponse>
}

/** Agent-side requests this adapter can serve; unimplemented ones are not registered. */
export interface Agent {
  initialize(params: InitializeRequest): Promise<InitializeResponse>
  newSession(params: NewSessionRequest): Promise<NewSessionResponse>
  prompt(params: PromptRequest): Promise<PromptResponse>
  cancel(params: CancelNotification): Promise<void>
  authenticate?(params: AuthenticateRequest): Promise<AuthenticateResponse | void>
  loadSession?(params: LoadSessionRequest): Promise<LoadSessionResponse | void>
  listSessions?(params: ListSessionsRequest): Promise<ListSessionsResponse>
  deleteSession?(params: DeleteSessionRequest): Promise<DeleteSessionResponse | void>
  resumeSession?(params: ResumeSessionRequest): Promise<ResumeSessionResponse>
  closeSession?(params: CloseSessionRequest): Promise<CloseSessionResponse | void>
  setSessionMode?(params: SetSessionModeRequest): Promise<SetSessionModeResponse | void>
  setSessionConfigOption?(params: SetSessionConfigOptionRequest): Promise<SetSessionConfigOptionResponse>
}

export interface ServedAgent<A extends Agent> {
  agent: A
  connection: AgentConnection
}

export function serveAgent<A extends Agent>(
  factory: (conn: AgentSideConnection) => A,
  stream: Stream,
  name = 'pi-acp'
): ServedAgent<A> {
  const open: { connection?: AgentConnection } = {}
  const client = () => {
    if (!open.connection) throw new Error('ACP connection is not open yet')
    return open.connection.client
  }

  const conn: AgentSideConnection = {
    sessionUpdate: params => client().notify(methods.client.session.update, params),
    requestPermission: params => client().request(methods.client.session.requestPermission, params)
  }

  const agent = factory(conn)
  const app = createAgentApp({ name })

  app.onRequest(methods.agent.initialize, ctx => agent.initialize(ctx.params))
  app.onRequest(methods.agent.session.new, ctx => agent.newSession(ctx.params))
  app.onRequest(methods.agent.session.prompt, ctx => agent.prompt(ctx.params))
  app.onNotification(methods.agent.session.cancel, ctx => agent.cancel(ctx.params))

  const a = agent
  if (a.authenticate) app.onRequest(methods.agent.authenticate, ctx => a.authenticate!(ctx.params))
  if (a.loadSession) app.onRequest(methods.agent.session.load, ctx => a.loadSession!(ctx.params))
  if (a.listSessions) app.onRequest(methods.agent.session.list, ctx => a.listSessions!(ctx.params))
  if (a.deleteSession) app.onRequest(methods.agent.session.delete, ctx => a.deleteSession!(ctx.params))
  if (a.resumeSession) app.onRequest(methods.agent.session.resume, ctx => a.resumeSession!(ctx.params))
  if (a.closeSession) app.onRequest(methods.agent.session.close, ctx => a.closeSession!(ctx.params))
  if (a.setSessionMode) app.onRequest(methods.agent.session.setMode, ctx => a.setSessionMode!(ctx.params))
  if (a.setSessionConfigOption) {
    app.onRequest(methods.agent.session.setConfigOption, ctx => a.setSessionConfigOption!(ctx.params))
  }

  open.connection = app.connect(stream)
  return { agent, connection: open.connection }
}
