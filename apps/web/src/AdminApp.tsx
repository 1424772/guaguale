import { FormEvent, useEffect, useMemo, useState } from 'react'
import { adminApi, ApiError, type AdminUser } from './api'
import { newIdempotencyKey } from './idempotency'
import './admin.css'

const coinFormatter = new Intl.NumberFormat('zh-CN')

function operationLabel(mode: BalanceMode) {
  if (mode === 'subtract') return '扣除'
  if (mode === 'set') return '设为'
  return '增加'
}

type BalanceMode = 'add' | 'subtract' | 'set'

export function AdminApp() {
  const [authenticated, setAuthenticated] = useState<boolean | null>(null)
  const [username, setUsername] = useState('admin')
  const [password, setPassword] = useState('')
  const [users, setUsers] = useState<AdminUser[]>([])
  const [query, setQuery] = useState('')
  const [selected, setSelected] = useState<AdminUser | null>(null)
  const [mode, setMode] = useState<BalanceMode>('add')
  const [amount, setAmount] = useState('20000')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  useEffect(() => {
    void loadUsers('')
  }, [])

  async function loadUsers(search: string) {
    try {
      const result = await adminApi.users(search)
      setUsers(result.users)
      setAuthenticated(true)
      setError('')
    } catch (requestError) {
      if (requestError instanceof ApiError && requestError.status === 401) {
        setAuthenticated(false)
        return
      }
      setAuthenticated(false)
      setError(requestError instanceof Error ? requestError.message : '管理员后台暂时无法连接')
    }
  }

  async function login(event: FormEvent) {
    event.preventDefault()
    setBusy(true)
    setError('')
    try {
      await adminApi.login(username.trim(), password)
      setPassword('')
      await loadUsers('')
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : '登录失败')
    } finally {
      setBusy(false)
    }
  }

  async function search(event: FormEvent) {
    event.preventDefault()
    setBusy(true)
    await loadUsers(query.trim())
    setBusy(false)
  }

  async function adjustBalance(event: FormEvent) {
    event.preventDefault()
    if (!selected) return
    const parsedAmount = Number(amount)
    if (!Number.isSafeInteger(parsedAmount) || parsedAmount < 0) {
      setError('请输入有效的整数金币数')
      return
    }
    setBusy(true)
    setError('')
    setNotice('')
    try {
      const idempotencyKey = newIdempotencyKey('admin-balance')
      const result = await adminApi.adjustBalance(selected.id, mode, parsedAmount, idempotencyKey)
      setUsers((current) => current.map((user) => user.id === result.user.id ? result.user : user))
      setSelected(result.user)
      setNotice(`${result.user.username} 的余额已${operationLabel(mode)}，当前为 ${coinFormatter.format(result.user.balance)} 金币`)
    } catch (requestError) {
      if (requestError instanceof ApiError && requestError.status === 401) {
        setAuthenticated(false)
      }
      setError(requestError instanceof Error ? requestError.message : '金币调整失败')
    } finally {
      setBusy(false)
    }
  }

  async function logout() {
    await adminApi.logout()
    setAuthenticated(false)
    setUsers([])
    setSelected(null)
  }

  const projectedBalance = useMemo(() => {
    if (!selected) return 0
    const value = Number(amount)
    if (!Number.isFinite(value)) return selected.balance
    if (mode === 'set') return value
    if (mode === 'subtract') return selected.balance - value
    return selected.balance + value
  }, [amount, mode, selected])

  if (authenticated === null) {
    return <main className="admin-loading"><span className="admin-spinner" />正在检查管理员登录状态…</main>
  }

  if (!authenticated) {
    return (
      <main className="admin-login-shell">
        <section className="admin-login-card">
          <div className="admin-brand-mark">刮</div>
          <p className="admin-kicker">SCRATCH DESK CONTROL</p>
          <h1>管理员后台</h1>
          <p className="admin-muted">账号管理、金币分配和操作审计</p>
          <form onSubmit={login}>
            <label>管理员账号<input autoComplete="username" value={username} onChange={(event) => setUsername(event.target.value)} /></label>
            <label>管理员密码<input autoComplete="current-password" type="password" value={password} onChange={(event) => setPassword(event.target.value)} /></label>
            {error && <p className="admin-error">{error}</p>}
            <button className="admin-primary" disabled={busy || !username.trim() || !password}>{busy ? '登录中…' : '进入后台'}</button>
          </form>
          <a className="admin-back-link" href="/">返回游戏</a>
        </section>
      </main>
    )
  }

  return (
    <main className="admin-shell">
      <header className="admin-topbar">
        <div><span className="admin-logo">刮</span><div><strong>刮个爽 · 管理后台</strong><small>账号与金币管理</small></div></div>
        <nav><a href="/">打开游戏</a><button onClick={() => void logout()}>退出后台</button></nav>
      </header>

      <section className="admin-content">
        <div className="admin-heading">
          <div><p className="admin-kicker">ACCOUNT OPERATIONS</p><h1>玩家账号</h1><p>搜索玩家并调整金币，所有变更均写入金币流水。</p></div>
          <form className="admin-search" onSubmit={search}>
            <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="输入完整或部分账号名" />
            <button disabled={busy}>搜索</button>
          </form>
        </div>

        {notice && <div className="admin-notice success">{notice}</div>}
        {error && <div className="admin-notice error">{error}</div>}

        <div className="admin-grid">
          <section className="admin-panel user-panel">
            <div className="admin-panel-title"><h2>账号列表</h2><span>{users.length} 个结果</span></div>
            <div className="admin-table-wrap">
              <table>
                <thead><tr><th>账号</th><th>金币余额</th><th>彩票</th><th>道具</th><th /></tr></thead>
                <tbody>
                  {users.map((user) => (
                    <tr className={selected?.id === user.id ? 'selected' : ''} key={user.id}>
                      <td><strong>{user.username}</strong><small>ID {user.id} · {new Date(user.createdAt).toLocaleDateString('zh-CN')}</small></td>
                      <td className="coins">{coinFormatter.format(user.balance)}</td>
                      <td>{user.ticketCount} 张</td>
                      <td><div className="admin-tags">{user.trashOwned && <span>垃圾桶</span>}{user.cardSlotsOwned && <span>卡槽</span>}{user.robotOwned && <span>机器人</span>}{!user.trashOwned && !user.cardSlotsOwned && !user.robotOwned && <em>暂无</em>}</div></td>
                      <td><button className="admin-row-action" onClick={() => { setSelected(user); setNotice(''); setError('') }}>分配金币</button></td>
                    </tr>
                  ))}
                  {users.length === 0 && <tr><td className="admin-empty" colSpan={5}>没有找到对应账号</td></tr>}
                </tbody>
              </table>
            </div>
          </section>

          <aside className="admin-panel balance-panel">
            {selected ? (
              <form onSubmit={adjustBalance}>
                <div className="selected-user"><span>{selected.username.slice(0, 1).toUpperCase()}</span><div><small>正在操作</small><h2>{selected.username}</h2><p>当前 {coinFormatter.format(selected.balance)} 金币</p></div></div>
                <fieldset>
                  <legend>调整方式</legend>
                  <div className="admin-segmented">
                    {(['add', 'subtract', 'set'] as BalanceMode[]).map((item) => <button className={mode === item ? 'active' : ''} type="button" key={item} onClick={() => setMode(item)}>{operationLabel(item)}</button>)}
                  </div>
                </fieldset>
                <label className="amount-label">金币数量<input inputMode="numeric" min="0" step="1" type="number" value={amount} onChange={(event) => setAmount(event.target.value)} /></label>
                <div className="quick-amounts">
                  {[20000, 100000, 1000000].map((value) => <button type="button" key={value} onClick={() => setAmount(String(value))}>+{coinFormatter.format(value)}</button>)}
                </div>
                <div className={`balance-preview ${projectedBalance < 0 ? 'invalid' : ''}`}><span>操作后余额</span><strong>{coinFormatter.format(projectedBalance)} <small>金币</small></strong></div>
                <button className="admin-primary" disabled={busy || projectedBalance < 0}>{busy ? '处理中…' : `确认${operationLabel(mode)}`}</button>
              </form>
            ) : (
              <div className="admin-placeholder"><span>¥</span><h2>选择一个账号</h2><p>点击列表里的“分配金币”开始操作</p></div>
            )}
          </aside>
        </div>
      </section>
    </main>
  )
}
