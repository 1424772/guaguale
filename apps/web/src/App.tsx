import { FormEvent, useEffect, useMemo, useState } from 'react'
import { ApiError, api, type Card, type DailyStatus, type Ticket, type User } from './api'
import { PlateCleaning } from './PlateCleaning'
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
  const [daily, setDaily] = useState<DailyStatus | null>(null)
  const [booting, setBooting] = useState(true)
  const [busy, setBusy] = useState(false)
  const [notice, setNotice] = useState('')
  const [activeTicket, setActiveTicket] = useState<Ticket | null>(null)
  const [scratchRequired, setScratchRequired] = useState(false)
  const [scratchComplete, setScratchComplete] = useState(false)
  const [dailyOpen, setDailyOpen] = useState(false)
  const [dailyBusy, setDailyBusy] = useState(false)
  const [wheelSpinning, setWheelSpinning] = useState(false)

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
    const [cardResponse, ticketResponse, dailyResponse] = await Promise.all([api.cards(), api.tickets(), api.daily()])
    setCards(cardResponse.cards)
    setTickets(ticketResponse.tickets)
    setDaily(dailyResponse.daily)
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
      setDaily(null)
      setActiveTicket(null)
    } catch (error) {
      setNotice(messageFrom(error))
    } finally {
      setBusy(false)
    }
  }

  async function claimDailyLogin() {
    if (dailyBusy) return
    setDailyBusy(true)
    try {
      const result = await api.claimDailyLogin()
      setUser(result.user)
      setCards((current) => updateUnlocks(current, result.user.balance))
      setDaily(result.daily)
      setNotice(result.idempotent ? '今天的登录奖励已经领取过了' : '领取成功，获得100金币')
    } catch (error) {
      setNotice(messageFrom(error))
    } finally {
      setDailyBusy(false)
    }
  }

  async function startPlate() {
    if (dailyBusy) return
    setDailyBusy(true)
    try {
      const result = await api.startPlate()
      setDaily(result.daily)
      setNotice(result.idempotent ? '继续清洗当前盘子' : `开始清洗第 ${result.daily.activePlate?.sequence ?? ''} 个盘子`)
    } catch (error) {
      setNotice(messageFrom(error))
    } finally {
      setDailyBusy(false)
    }
  }

  async function completePlate(actionId: string) {
    if (dailyBusy) return
    setDailyBusy(true)
    try {
      const result = await api.completePlate(actionId)
      setUser(result.user)
      setCards((current) => updateUnlocks(current, result.user.balance))
      setDaily(result.daily)
      setNotice(result.idempotent ? '这个盘子已经结算' : '盘子洗好了，获得5金币')
    } catch (error) {
      if (!(error instanceof ApiError && error.code === 'too_early')) {
        setNotice(messageFrom(error))
      }
    } finally {
      setDailyBusy(false)
    }
  }

  async function spinWheel() {
    if (dailyBusy || wheelSpinning || !daily || daily.wheelUsed || daily.wheelPool.length === 0) return
    setDailyBusy(true)
    setWheelSpinning(true)
    try {
      const result = await api.spinWheel()
      await new Promise((resolve) => window.setTimeout(resolve, 1800))
      setUser(result.user)
      setDaily(result.daily)
      setTickets((current) => [result.ticket, ...current.filter((ticket) => ticket.id !== result.ticket.id)])
      setNotice(`转盘获得《${result.ticket.cardName}》，已放到桌面`)
    } catch (error) {
      setNotice(messageFrom(error))
    } finally {
      setWheelSpinning(false)
      setDailyBusy(false)
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
          <button type="button" onClick={() => setDailyOpen(true)}>今日任务</button>
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
              <button className="tier-row" key={card.code} type="button" disabled={!card.implemented || !card.unlocked || busy} onClick={() => purchase(card)}>
                <span className="tier-number">{index + 2}</span>
                <div><strong>{card.name}</strong><small>{coinFormatter.format(card.price)} 金币门槛</small></div>
                <span className={card.unlocked ? 'unlocked' : 'locked'}>{!card.implemented ? '待开发' : card.unlocked ? '购买' : '未解锁'}</span>
              </button>
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
              <ScratchCard cardName={activeTicket.cardName} symbols={activeTicket.symbols ?? []} onComplete={() => setScratchComplete(true)} />
            ) : (
              <ResultSymbols cardName={activeTicket.cardName} symbols={activeTicket.symbols ?? []} />
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

      {dailyOpen && daily && user && (
        <DailyTasksDialog
          daily={daily}
          balance={user.balance}
          busy={dailyBusy}
          spinning={wheelSpinning}
          onClose={() => setDailyOpen(false)}
          onClaimLogin={claimDailyLogin}
          onStartPlate={startPlate}
          onCompletePlate={completePlate}
          onSpin={spinWheel}
        />
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

function DailyTasksDialog({
  daily,
  balance,
  busy,
  spinning,
  onClose,
  onClaimLogin,
  onStartPlate,
  onCompletePlate,
  onSpin,
}: {
  daily: DailyStatus
  balance: number
  busy: boolean
  spinning: boolean
  onClose: () => void
  onClaimLogin: () => void
  onStartPlate: () => void
  onCompletePlate: (actionId: string) => void
  onSpin: () => void
}) {
  const platesDone = daily.platesCompleted >= daily.plateLimit
  const wheelUnavailable = daily.wheelPool.length === 0
  const wheelColors = ['#d7b45a', '#407b62', '#ae7540', '#365b78', '#76518e', '#9c4747']
  let wheelOffset = 0
  const wheelStops = daily.wheelPool.map((item, index) => {
    const start = wheelOffset
    wheelOffset += item.basisPoint / 100
    return `${wheelColors[index % wheelColors.length]} ${start}% ${wheelOffset}%`
  })
  const wheelBackground = wheelStops.length > 0
    ? `conic-gradient(from -20deg, ${wheelStops.join(', ')})`
    : 'conic-gradient(#4b514c 0 100%)'

  return (
    <div className="modal-backdrop" role="presentation" onMouseDown={(event) => {
      if (event.target === event.currentTarget && !spinning) onClose()
    }}>
      <section className="daily-dialog" role="dialog" aria-modal="true" aria-labelledby="daily-title">
        <button className="close-button" type="button" onClick={onClose} disabled={spinning} aria-label="关闭">×</button>
        <header className="daily-header">
          <span className="eyebrow">DAILY RECOVERY</span>
          <h1 id="daily-title">今日任务</h1>
          <p>{daily.date} · 每日零点刷新，未使用次数不累计</p>
        </header>

        <div className="daily-grid">
          <article className={`task-card ${daily.loginClaimed ? 'task-complete' : ''}`}>
            <div className="task-icon" aria-hidden="true">☀️</div>
            <div className="task-copy">
              <span>每日登录</span>
              <h2>领取 100 金币</h2>
              <p>每天一次，为下一轮游戏提供基础恢复资金。</p>
            </div>
            <button className="task-button" type="button" onClick={onClaimLogin} disabled={busy || daily.loginClaimed}>
              {daily.loginClaimed ? '今日已领取 ✓' : '领取奖励'}
            </button>
          </article>

          <article className={`task-card ${platesDone ? 'task-complete' : ''}`}>
            <div className="task-icon" aria-hidden="true">🍽️</div>
            <div className="task-copy">
              <span>洗盘子</span>
              <h2>每个 5 金币</h2>
              <p>每天最多洗 {daily.plateLimit} 个；按住并擦掉盘面残渣，清洁达标后结算。</p>
              <div className="task-progress" aria-label={`已完成${daily.platesCompleted}个，共${daily.plateLimit}个`}>
                <span style={{ width: `${(daily.platesCompleted / daily.plateLimit) * 100}%` }} />
              </div>
              <small>{daily.platesCompleted} / {daily.plateLimit} 已完成</small>
            </div>
            {daily.activePlate ? (
              <div className="plate-active-label"><span className="plate-bubble">◌</span>正在清洗第 {daily.activePlate.sequence} 个盘子</div>
            ) : (
              <button className="task-button" type="button" onClick={onStartPlate} disabled={busy || platesDone}>
                {platesDone ? '今日已完成 ✓' : `清洗第 ${daily.platesCompleted + 1} 个`}
              </button>
            )}
          </article>
        </div>

        {daily.activePlate && (
          <PlateCleaning
            actionId={daily.activePlate.id}
            sequence={daily.activePlate.sequence}
            plateLimit={daily.plateLimit}
            availableAt={daily.activePlate.availableAt}
            busy={busy}
            onComplete={onCompletePlate}
          />
        )}

        <section className="wheel-section">
          <div className="wheel-visual" aria-label="每日免费卡转盘">
            <div className="wheel-pointer" aria-hidden="true">▼</div>
            <div className={`wheel-disc ${spinning ? 'wheel-spinning' : ''}`} style={{ background: wheelBackground }}>
              <div className="wheel-hub"><span>每日</span><strong>1 次</strong></div>
            </div>
          </div>

          <div className="wheel-copy">
            <span className="eyebrow">FREE CARD WHEEL</span>
            <h2>免费刮奖卡转盘</h2>
            <p>只会抽到当前 {coinFormatter.format(balance)} 金币可购买范围内的第 1–6 款卡；第 7、8 款永不参与。</p>
            <div className="wheel-pool">
              {daily.wheelPool.map((item) => (
                <div className="pool-row" key={item.cardCode}>
                  <span>{item.cardName}<small>{coinFormatter.format(item.price)} 金币门槛</small></span>
                  <strong>{(item.basisPoint / 100).toFixed(2)}%</strong>
                </div>
              ))}
              {wheelUnavailable && <div className="pool-empty">余额不足 50 金币，先领取登录奖励即可恢复转盘资格。</div>}
              <div className="excluded-row"><span>永恒彩钻、放手一博</span><strong>不参与</strong></div>
            </div>
            {daily.wheelUsed ? (
              <div className="wheel-result"><span>今日结果</span><strong>《{daily.wheelCardName ?? '免费刮奖卡'}》</strong><small>卡片已经放到桌面</small></div>
            ) : (
              <button className="gold-button wheel-button" type="button" onClick={onSpin} disabled={busy || spinning || wheelUnavailable}>
                {spinning ? '转盘转动中…' : wheelUnavailable ? '至少持有 50 金币后可转动' : '免费转动 · 今日 1/1'}
              </button>
            )}
          </div>
        </section>
      </section>
    </div>
  )
}

function ResultSymbols({ cardName, symbols }: { cardName: string; symbols: string[] }) {
  const emoji: Record<string, string> = {
    '狗头金币': '🐶', '钞票': '💵', '碎钻石': '💎', '钞票堆': '💰',
    '辣条': '🌶️', '可乐': '🥤', '冰棍': '🧊', '雪糕': '🍦', '玩具车': '🚗', '小电视': '📺', '游戏机': '🎮',
    '弹珠': '🔵', '拳套': '🥊', '赛车': '🏎️', '飞机': '✈️', '街机皇冠': '👑',
    '青晶簇': '🔷', '红晶簇': '🔶', '紫晶簇': '💜', '金色矿石': '🪨',
    '海神王冠': '👑', '黄金宝箱': '🧰', '珍珠贝': '🦪', '生锈船锚': '⚓', '漂流瓶': '🍾', '破皮靴': '🥾', '海草团': '🌿', '空网': '🕸️',
  }
  return (
    <div className="result-ticket">
      <span>{cardName}</span>
      <div className={`result-symbols count-${symbols.length}`}>{symbols.map((symbol, index) => {
        const baseSymbol = symbol.replace(/^目标·/, '')
        const fuelValue = symbol.match(/^燃料 (\d)$/)?.[1]
        return <strong key={`${symbol}-${index}`}>{fuelValue ? `⛽${fuelValue}` : emoji[baseSymbol] ?? '✦'}<small>{symbol.replace('目标·', '目标：')}</small></strong>
      })}</div>
    </div>
  )
}

function messageFrom(error: unknown) {
  return error instanceof Error ? error.message : '发生未知错误，请稍后重试'
}
