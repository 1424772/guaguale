import { FormEvent, useEffect, useMemo, useState } from 'react'
import { ApiError, api, type Card, type Ticket, type User } from './api'
import { ScratchCard } from './ScratchCard'
import ticketArtwork from './assets/concepts/lingqian-ticket-play-v1.webp'

const coinFormatter = new Intl.NumberFormat('zh-CN')

function updateUnlocks(cards: Card[], balance: number) {
  return cards.map((card) => ({ ...card, unlocked: balance >= card.price }))
}

export function App() {
  const [user, setUser] = useState<User | null>(null)
  const [cards, setCards] = useState<Card[]>([])
  const [tickets, setTickets] = useState<Ticket[]>([])
  const [booting, setBooting] = useState(true)
  const [busy, setBusy] = useState(false)
  const [notice, setNotice] = useState('')
  const [activeTicket, setActiveTicket] = useState<Ticket | null>(null)
  const [scratchRequired, setScratchRequired] = useState(false)
  const [scratchComplete, setScratchComplete] = useState(false)

  useEffect(() => {
    void bootstrap()
  }, [])

  async function bootstrap() {
    try {
      const me = await api.me()
      setUser(me.user)
      await loadGameData()
    } catch (error) {
      if (!(error instanceof ApiError && error.status === 401)) {
        setNotice(messageFrom(error))
      }
    } finally {
      setBooting(false)
    }
  }

  async function loadGameData() {
    const [cardResponse, ticketResponse] = await Promise.all([api.cards(), api.tickets()])
    setCards(cardResponse.cards)
    setTickets(ticketResponse.tickets)
  }

  async function handleAuthenticated(nextUser: User) {
    setUser(nextUser)
    setNotice('欢迎来到刮刮乐桌面')
    await loadGameData()
  }

  async function handleLogout() {
    setBusy(true)
    try {
      await api.logout()
      setUser(null)
      setCards([])
      setTickets([])
      setActiveTicket(null)
    } catch (error) {
      setNotice(messageFrom(error))
    } finally {
      setBusy(false)
    }
  }

  async function purchase(card: Card) {
    if (busy || !card.implemented || !card.unlocked) return
    setBusy(true)
    setNotice('')
    try {
      const result = await api.purchase(card.code, crypto.randomUUID())
      setUser(result.user)
      setCards((current) => updateUnlocks(current, result.user.balance))
      setTickets((current) => [result.ticket, ...current.filter((ticket) => ticket.id !== result.ticket.id)])
      setNotice('购买成功，卡片已放到桌面')
    } catch (error) {
      setNotice(messageFrom(error))
    } finally {
      setBusy(false)
    }
  }

  async function openTicket(ticket: Ticket) {
    if (busy) return
    setNotice('')
    if (ticket.state !== 'purchased') {
      setActiveTicket(ticket)
      setScratchRequired(false)
      setScratchComplete(true)
      return
    }
    setBusy(true)
    try {
      const response = await api.scratch(ticket.id)
      setTickets((current) => current.map((item) => item.id === response.ticket.id ? response.ticket : item))
      setActiveTicket(response.ticket)
      setScratchRequired(true)
      setScratchComplete(false)
    } catch (error) {
      setNotice(messageFrom(error))
    } finally {
      setBusy(false)
    }
  }

  async function redeem() {
    if (!activeTicket || !activeTicket.reward || busy) return
    setBusy(true)
    try {
      const result = await api.redeem(activeTicket.id)
      setUser(result.user)
      setCards((current) => updateUnlocks(current, result.user.balance))
      setTickets((current) => current.map((ticket) => ticket.id === result.ticket.id ? result.ticket : ticket))
      setActiveTicket(result.ticket)
      setNotice(`兑奖成功，获得 ${coinFormatter.format(result.ticket.reward ?? 0)} 金币`)
    } catch (error) {
      setNotice(messageFrom(error))
    } finally {
      setBusy(false)
    }
  }

  if (booting) {
    return <div className="loading-screen"><span className="spinner" />正在打开桌面…</div>
  }

  if (!user) {
    return <AuthScreen onAuthenticated={handleAuthenticated} />
  }

  const firstCard = cards.find((card) => card.code === 'lingqian-ticket')

  return (
    <main className="game-shell">
      <header className="topbar">
        <div className="identity">
          <div className="avatar" aria-hidden="true">🐶</div>
          <div><small>{user.username}</small><strong>金币 {coinFormatter.format(user.balance)}</strong></div>
        </div>
        <nav aria-label="主要功能">
          <button type="button">今日任务</button>
          <button type="button">商店</button>
          <button type="button">排行榜</button>
          <button type="button" onClick={handleLogout} disabled={busy}>退出</button>
        </nav>
      </header>

      {notice && <div className="notice" role="status">{notice}</div>}

      <section className="desk">
        <aside className="catalog-panel">
          <div className="panel-title"><span>购卡托盘</span><small>余额达到售价即可购买</small></div>
          {firstCard && (
            <article className="featured-card">
              <img src={ticketArtwork} alt="零钱小票卡面概念图" />
              <div className="featured-copy">
                <div><span className="tag">已开放</span><h2>{firstCard.name}</h2></div>
                <p>三格中出现两格相同即可获得对应奖励，三格相同奖励翻倍。</p>
                <button
                  type="button"
                  className="gold-button"
                  disabled={busy || !firstCard.unlocked}
                  onClick={() => purchase(firstCard)}
                >
                  {firstCard.unlocked ? `购买 · ${coinFormatter.format(firstCard.price)} 金币` : `还差 ${coinFormatter.format(firstCard.price - user.balance)} 金币`}
                </button>
              </div>
            </article>
          )}

          <div className="tier-list" aria-label="后续卡片">
            {cards.filter((card) => card.code !== 'lingqian-ticket').map((card, index) => (
              <div className="tier-row" key={card.code}>
                <span className="tier-number">{index + 2}</span>
                <div><strong>{card.name}</strong><small>{coinFormatter.format(card.price)} 金币门槛</small></div>
                <span className={card.unlocked ? 'unlocked' : 'locked'}>{card.unlocked ? '待开放' : '未解锁'}</span>
              </div>
            ))}
          </div>
        </aside>

        <section className="table-area" aria-labelledby="ticket-heading">
          <div className="table-heading">
            <div><span className="eyebrow">MY DESK</span><h1 id="ticket-heading">我的桌面</h1></div>
            <p>未兑奖卡会一直保留</p>
          </div>

          {tickets.length === 0 ? (
            <div className="empty-desk">
              <div>✦</div>
              <h2>桌面还是空的</h2>
              <p>购买第一张《零钱小票》，开始完整刮奖流程。</p>
            </div>
          ) : (
            <div className="ticket-grid">
              {tickets.map((ticket) => (
                <button
                  type="button"
                  className={`desk-ticket state-${ticket.state}`}
                  key={ticket.id}
                  onClick={() => openTicket(ticket)}
                >
                  <span className="ticket-stamp">{ticket.cardName}</span>
                  <strong>{ticket.state === 'purchased' ? '等待刮开' : ticket.state === 'redeemed' ? '已兑奖' : '查看结果'}</strong>
                  <small>{new Date(ticket.createdAt).toLocaleString('zh-CN', { hour12: false })}</small>
                </button>
              ))}
            </div>
          )}
        </section>
      </section>

      {activeTicket && (
        <div className="modal-backdrop" role="presentation" onMouseDown={(event) => {
          if (event.target === event.currentTarget) setActiveTicket(null)
        }}>
          <section className="scratch-dialog" role="dialog" aria-modal="true" aria-label="刮奖">
            <button className="close-button" type="button" onClick={() => setActiveTicket(null)} aria-label="关闭">×</button>
            {scratchRequired ? (
              <ScratchCard symbols={activeTicket.symbols ?? []} onComplete={() => setScratchComplete(true)} />
            ) : (
              <ResultSymbols symbols={activeTicket.symbols ?? []} />
            )}
            {scratchComplete && (
              <div className={`result-box ${activeTicket.reward ? 'winner' : 'loser'}`}>
                <small>本张结果</small>
                <h2>{activeTicket.reward ? `获得 ${coinFormatter.format(activeTicket.reward)} 金币` : '未中奖'}</h2>
                <p>{activeTicket.reward ? '把中奖卡放入兑奖区即可入账。' : '未中奖卡将继续留在桌面，后续可丢入垃圾桶。'}</p>
                {activeTicket.state === 'scratched' && Boolean(activeTicket.reward) && (
                  <button type="button" className="gold-button" onClick={redeem} disabled={busy}>拖入兑奖区 · 立即兑奖</button>
                )}
                {activeTicket.state === 'redeemed' && <span className="redeemed-badge">已经兑奖</span>}
              </div>
            )}
          </section>
        </div>
      )}
    </main>
  )
}

function AuthScreen({ onAuthenticated }: { onAuthenticated: (user: User) => Promise<void> }) {
  const [mode, setMode] = useState<'register' | 'login'>('register')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [ageConfirmed, setAgeConfirmed] = useState(false)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const valid = useMemo(() => {
    const usernameLength = [...username.trim()].length
    return usernameLength >= 3 && usernameLength <= 32 && new TextEncoder().encode(password).length >= 8 && (mode === 'login' || ageConfirmed)
  }, [ageConfirmed, mode, password, username])

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (!valid || busy) return
    setBusy(true)
    setError('')
    try {
      const result = mode === 'register'
        ? await api.register(username.trim(), password)
        : await api.login(username.trim(), password)
      await onAuthenticated(result.user)
    } catch (caught) {
      setError(messageFrom(caught))
    } finally {
      setBusy(false)
    }
  }

  return (
    <main className="auth-shell">
      <section className="auth-story">
        <span className="eyebrow">PURE VIRTUAL COINS</span>
        <h1>一张小票，<br />看清概率的长期结果。</h1>
        <p>这是一个只使用虚拟金币的刮刮乐游戏：不能充值、不能提现、不能交易，也没有现金或实物奖励。</p>
        <ul>
          <li>新账号获得 1,000 虚拟金币</li>
          <li>所有开奖结果由服务端提前锁定</li>
          <li>历史页将永久保留投入与兑奖记录</li>
        </ul>
      </section>
      <section className="auth-card">
        <div className="auth-tabs">
          <button type="button" className={mode === 'register' ? 'active' : ''} onClick={() => setMode('register')}>注册</button>
          <button type="button" className={mode === 'login' ? 'active' : ''} onClick={() => setMode('login')}>登录</button>
        </div>
        <form onSubmit={submit}>
          <label>用户名<input autoComplete="username" value={username} onChange={(event) => setUsername(event.target.value)} placeholder="3–32个字符" /></label>
          <label>密码<input type="password" autoComplete={mode === 'register' ? 'new-password' : 'current-password'} value={password} onChange={(event) => setPassword(event.target.value)} placeholder="至少8个字符" /></label>
          {mode === 'register' && (
            <label className="age-check"><input type="checkbox" checked={ageConfirmed} onChange={(event) => setAgeConfirmed(event.target.checked)} />我已满16周岁，并理解游戏只使用虚拟金币</label>
          )}
          {error && <p className="form-error" role="alert">{error}</p>}
          <button className="gold-button" type="submit" disabled={!valid || busy}>{busy ? '请稍候…' : mode === 'register' ? '创建账号并领取1,000金币' : '进入桌面'}</button>
        </form>
      </section>
    </main>
  )
}

function ResultSymbols({ symbols }: { symbols: string[] }) {
  const emoji: Record<string, string> = { '狗头金币': '🐶', '钞票': '💵', '碎钻石': '💎', '钞票堆': '💰' }
  return (
    <div className="result-ticket">
      <span>零钱小票</span>
      <div>{symbols.map((symbol, index) => <strong key={`${symbol}-${index}`}>{emoji[symbol] ?? '✦'}<small>{symbol}</small></strong>)}</div>
    </div>
  )
}

function messageFrom(error: unknown) {
  return error instanceof Error ? error.message : '发生未知错误，请稍后重试'
}
