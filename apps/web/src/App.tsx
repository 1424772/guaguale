import { CSSProperties, DragEvent as ReactDragEvent, FormEvent, KeyboardEvent as ReactKeyboardEvent, PointerEvent as ReactPointerEvent, useEffect, useMemo, useRef, useState } from 'react'
import { ApiError, api, type Card, type DailyStatus, type FanCardEvent, type FanStatus, type HistoryEvent, type Leaderboard, type RobotStatus, type ShopItem, type ShopStatus, type Ticket, type User } from './api'
import { DeskTicket, TicketResultSurface, type DeskPlacement } from './DeskTicket'
import { HistoryDialog } from './HistoryDialog'
import { LeaderboardDialog } from './LeaderboardDialog'
import { PlateCleaning } from './PlateCleaning'
import { getSymbolVisual, scratchLayoutFor, ScratchCard, symbolClassName, symbolStateClassNames, TicketSymbolGlyph } from './ScratchCard'
import { ShopDialog } from './ShopDialog'
import { RobotDialog } from './RobotDialog'
import { TicketScratchCoating, usesFullCoatingPreview } from './TicketScratchCoating'
import { ticketArtworkFor, unopenedTicketArtworkFor } from './ticketArtwork'
import { newIdempotencyKey } from './idempotency'

const coinFormatter = new Intl.NumberFormat('zh-CN')

function updateUnlocks(cards: Card[], balance: number) {
  return cards.map((card) => ({ ...card, unlocked: balance >= card.price }))
}

function savedScratchProgress() {
  try {
    return JSON.parse(window.localStorage.getItem('guaguale:scratch-progress') ?? '{}') as Record<string, number>
  } catch {
    return {}
  }
}

function savedScratchSnapshots() {
  try {
    return JSON.parse(window.localStorage.getItem('guaguale:scratch-snapshots') ?? '{}') as Record<string, string>
  } catch {
    return {}
  }
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
  const [shopOpen, setShopOpen] = useState(false)
  const [shopFocus, setShopFocus] = useState<ShopItem['code']>('luck')
  const [shop, setShop] = useState<ShopStatus | null>(null)
  const [catalogOpen, setCatalogOpen] = useState(false)
  const [robot, setRobot] = useState<RobotStatus | null>(null)
  const [robotOpen, setRobotOpen] = useState(false)
  const [fan, setFan] = useState<FanStatus | null>(null)
  const [fanRiskOpen, setFanRiskOpen] = useState(false)
  const [fanRiskPending, setFanRiskPending] = useState(false)
  const [fanHolding, setFanHolding] = useState(false)
  const [fanBusy, setFanBusy] = useState(false)
  const [fanActions, setFanActions] = useState<Record<string, FanCardEvent['action']>>({})
  const [leaderboardOpen, setLeaderboardOpen] = useState(false)
  const [leaderboard, setLeaderboard] = useState<Leaderboard | null>(null)
  const [leaderboardLoading, setLeaderboardLoading] = useState(false)
  const [leaderboardError, setLeaderboardError] = useState('')
  const [historyOpen, setHistoryOpen] = useState(false)
  const [historyEvents, setHistoryEvents] = useState<HistoryEvent[]>([])
  const [historyLoading, setHistoryLoading] = useState(false)
  const [historyError, setHistoryError] = useState('')
  const [dailyBusy, setDailyBusy] = useState(false)
  const [wheelSpinning, setWheelSpinning] = useState(false)
  const [pendingDiscard, setPendingDiscard] = useState<Ticket | null>(null)
  const [discardingTicketId, setDiscardingTicketId] = useState<string | null>(null)
  const [redeemingTicketId, setRedeemingTicketId] = useState<string | null>(null)
  const [robotEjectedTicketId, setRobotEjectedTicketId] = useState<string | null>(null)
  const [robotIntakeTicket, setRobotIntakeTicket] = useState<Ticket | null>(null)
  const [robotOutputTicket, setRobotOutputTicket] = useState<Ticket | null>(null)
  const [coinBurst, setCoinBurst] = useState<{ key: number; reward: number } | null>(null)
  const [scratchProgress, setScratchProgress] = useState<Record<string, number>>(savedScratchProgress)
  const [scratchSnapshots, setScratchSnapshots] = useState<Record<string, string>>(savedScratchSnapshots)
  const [trayDragging, setTrayDragging] = useState(false)
  const discardTimerRef = useRef<number | null>(null)
  const discardAnimationTimerRef = useRef<number | null>(null)
  const deskRef = useRef<HTMLDivElement>(null)
  const redeemZoneRef = useRef<HTMLDivElement>(null)
  const trashZoneRef = useRef<HTMLDivElement>(null)
  const robotZoneRef = useRef<HTMLButtonElement>(null)
  const slotRefs = useRef<Array<HTMLDivElement | null>>([])
  const robotTickingRef = useRef(false)
  const robotIntakeTimerRef = useRef<number | null>(null)
  const robotOutputTimerRef = useRef<number | null>(null)
  const fanHoldTimerRef = useRef<number | null>(null)

  useEffect(() => {
    void bootstrap()
  }, [])

  useEffect(() => {
    if (!robot?.owned || robot.queue.length === 0) return
    let stopped = false
    async function tick() {
      if (stopped || document.hidden || robotTickingRef.current) return
      robotTickingRef.current = true
      try {
        const result = await api.tickRobot()
        if (stopped) return
        setUser(result.user)
        setCards((current) => updateUnlocks(current, result.user.balance))
        setRobot(result.robot)
        if (result.event) {
          const finished = result.event.ticket
          setTickets((current) => current.map((ticket) => ticket.id === finished.id ? finished : ticket))
          setRobotOutputTicket(finished)
          setRobotEjectedTicketId(finished.id)
          if (robotOutputTimerRef.current !== null) window.clearTimeout(robotOutputTimerRef.current)
          robotOutputTimerRef.current = window.setTimeout(() => {
            setRobotOutputTicket((current) => current?.id === finished.id ? null : current)
            setRobotEjectedTicketId((current) => current === finished.id ? null : current)
            robotOutputTimerRef.current = null
          }, 1900)
          setNotice(finished.reward
            ? `${finished.prizeTier === 'jackpot' ? '🎉 命中头奖！' : ''}机器人吐出《${finished.cardName}》，中奖 ${coinFormatter.format(finished.reward)} 金币，请手动兑奖`
            : `机器人吐出《${finished.cardName}》，本张未中奖`)
        }
      } catch (error) {
        if (!stopped) setNotice(messageFrom(error))
      } finally {
        robotTickingRef.current = false
      }
    }
    void tick()
    const timer = window.setInterval(() => void tick(), 1000)
    return () => {
      stopped = true
      window.clearInterval(timer)
    }
  }, [robot?.owned, robot?.queue.length])

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
    const [cardResponse, ticketResponse, dailyResponse, robotResponse, fanResponse] = await Promise.all([api.cards(), api.tickets(), api.daily(), api.robot(), api.fan()])
    setCards(cardResponse.cards)
    setTickets(ticketResponse.tickets)
    setDaily(dailyResponse.daily)
    setRobot(robotResponse.robot)
    setFan(fanResponse.fan)
  }

  async function handleAuthenticated(nextUser: User) {
    setUser(nextUser)
    setNotice('欢迎来到刮刮乐桌面')
    await loadGameData()
  }

  async function handleLogout() {
    setBusy(true)
    try {
      if (discardTimerRef.current !== null) window.clearTimeout(discardTimerRef.current)
      if (discardAnimationTimerRef.current !== null) window.clearTimeout(discardAnimationTimerRef.current)
      if (fanHoldTimerRef.current !== null) window.clearTimeout(fanHoldTimerRef.current)
      if (robotIntakeTimerRef.current !== null) window.clearTimeout(robotIntakeTimerRef.current)
      if (robotOutputTimerRef.current !== null) window.clearTimeout(robotOutputTimerRef.current)
      discardTimerRef.current = null
      discardAnimationTimerRef.current = null
      robotIntakeTimerRef.current = null
      robotOutputTimerRef.current = null
      setPendingDiscard(null)
      setRobotIntakeTicket(null)
      setRobotOutputTicket(null)
      await api.logout()
      setUser(null)
      setCards([])
      setTickets([])
      setDaily(null)
      setShop(null)
      setRobot(null)
      setFan(null)
      setRobotOpen(false)
      setFanRiskOpen(false)
      setFanActions({})
      setShopOpen(false)
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
      const placed = await placeNewTicketOnDesk(result.ticket)
      setTickets((current) => [placed, ...current.filter((ticket) => ticket.id !== placed.id)])
      setNotice(`转盘获得《${placed.cardName}》，已放到桌面`)
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
      const result = await api.purchase(card.code, newIdempotencyKey('card'))
      setUser(result.user)
      setCards((current) => updateUnlocks(current, result.user.balance))
      const placed = await placeNewTicketOnDesk(result.ticket)
      setTickets((current) => [placed, ...current.filter((ticket) => ticket.id !== placed.id)])
      setCatalogOpen(false)
      setNotice(`购买成功，《${placed.cardName}》已放到桌面`)
    } catch (error) {
      setNotice(messageFrom(error))
    } finally {
      setBusy(false)
    }
  }

  async function openShop(focus: ShopItem['code'] = 'luck') {
    if (busy) return
    setBusy(true)
    try {
      const result = await api.shop()
      setShop(result.shop)
      setShopFocus(focus)
      setShopOpen(true)
    } catch (error) {
      setNotice(messageFrom(error))
    } finally {
      setBusy(false)
    }
  }

  async function openLeaderboard() {
    setLeaderboardOpen(true)
    setLeaderboardLoading(true)
    setLeaderboardError('')
    try {
      const response = await api.leaderboard()
      setLeaderboard(response.leaderboard)
    } catch (error) {
      setLeaderboardError(messageFrom(error))
    } finally {
      setLeaderboardLoading(false)
    }
  }

  async function openHistory() {
    setHistoryOpen(true)
    setHistoryLoading(true)
    setHistoryError('')
    try {
      const response = await api.history()
      setHistoryEvents(response.events)
    } catch (error) {
      setHistoryError(messageFrom(error))
    } finally {
      setHistoryLoading(false)
    }
  }

  async function upgradeItem(item: ShopItem) {
    if (busy || !item.nextPrice) return
    setBusy(true)
    try {
      const result = await api.upgradeItem(item.code, newIdempotencyKey('item'))
      setUser(result.user)
      setCards((current) => updateUnlocks(current, result.user.balance))
      setShop(result.shop)
      if (item.code.startsWith('robot')) {
        const robotResponse = await api.robot()
        setRobot(robotResponse.robot)
      }
      if (item.code === 'fan') {
        const fanResponse = await api.fan()
        setFan(fanResponse.fan)
      }
      const updated = result.shop.items.find((candidate) => candidate.code === result.itemCode)
      setNotice(item.maxLevel === 1 ? `${updated?.name ?? item.name}购买成功，已永久开放` : `${updated?.name ?? item.name}已升级到 ${updated?.level ?? item.level + 1} 级`)
    } catch (error) {
      setNotice(messageFrom(error))
      try {
        const [me, currentShop] = await Promise.all([api.me(), api.shop()])
        setUser(me.user)
        setShop(currentShop.shop)
        setCards((current) => updateUnlocks(current, me.user.balance))
      } catch {
        // A later normal refresh will reconcile account state.
      }
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
      const response = await api.revealTicket(ticket.id)
      setActiveTicket(response.ticket)
      setScratchRequired(true)
      setScratchComplete(false)
    } catch (error) {
      setNotice(messageFrom(error))
    } finally {
      setBusy(false)
    }
  }

  async function completeScratch(ticketId: string) {
    if (busy) return
    setBusy(true)
    try {
      const response = await api.scratch(ticketId)
      setTickets((current) => current.map((item) => item.id === response.ticket.id ? response.ticket : item))
      setActiveTicket((current) => current?.id === response.ticket.id ? response.ticket : current)
      rememberScratchProgress(ticketId, 100)
      setScratchComplete(true)
      if (response.ticket.prizeTier === 'jackpot') {
        setNotice(`🎉 命中头奖！《${response.ticket.cardName}》获得 ${coinFormatter.format(response.ticket.reward ?? 0)} 金币`)
      }
    } catch (error) {
      setNotice(messageFrom(error))
    } finally {
      setBusy(false)
    }
  }

  async function redeem() {
    if (!activeTicket || !activeTicket.reward || busy) return
    await redeemTicket(activeTicket)
  }

  async function redeemTicket(ticket: Ticket) {
    if (!ticket.reward || busy) return
    setBusy(true)
    setActiveTicket(null)
    setRedeemingTicketId(ticket.id)
    try {
      await new Promise((resolve) => window.setTimeout(resolve, 1050))
      const result = await api.redeem(ticket.id)
      setUser(result.user)
      setCards((current) => updateUnlocks(current, result.user.balance))
      setTickets((current) => current.filter((item) => item.id !== result.ticket.id))
      setCoinBurst({ key: Date.now(), reward: result.ticket.reward ?? 0 })
      window.setTimeout(() => setCoinBurst(null), 1500)
      setNotice(`兑奖成功，获得 ${coinFormatter.format(result.ticket.reward ?? 0)} 金币`)
    } catch (error) {
      setNotice(messageFrom(error))
    } finally {
      setRedeemingTicketId(null)
      setBusy(false)
    }
  }

  function rememberScratchProgress(ticketId: string, progress: number) {
    setScratchProgress((current) => {
      const next = { ...current, [ticketId]: progress }
      try {
        window.localStorage.setItem('guaguale:scratch-progress', JSON.stringify(next))
      } catch {
        // The live UI remains accurate even when storage is unavailable.
      }
      return next
    })
  }

  function rememberScratchSnapshot(ticketId: string, snapshot: string) {
    setScratchSnapshots((current) => {
      const next = { ...current, [ticketId]: snapshot }
      try {
        window.localStorage.setItem('guaguale:scratch-snapshots', JSON.stringify(next))
      } catch {
        // Keep the current page accurate if persistence is unavailable.
      }
      return next
    })
  }

  function nextDeskPlacement(offset = 0) {
    const deskCount = tickets.filter((item) => item.location === 'desk').length + offset
    const column = deskCount % 3
    const row = Math.floor(deskCount / 3) % 3
    return {
      location: 'desk' as const,
      deskX: .40 + column * .16,
      deskY: .33 + row * .19,
      rotation: ((deskCount * 7) % 11) - 5,
      zIndex: Math.max(1, ...tickets.map((item) => item.zIndex)) + 1 + offset,
    }
  }

  async function placeNewTicketOnDesk(ticket: Ticket) {
    const placement = nextDeskPlacement()
    try {
      const result = await api.placeTicket(ticket.id, placement)
      return result.ticket
    } catch {
      return { ...ticket, ...placement }
    }
  }

  async function updateTicketPlacement(ticket: Ticket, placement: {
    location: 'desk' | 'slot'
    deskX: number
    deskY: number
    rotation: number
    zIndex: number
    slotIndex?: number
  }) {
    setTickets((current) => current.map((item) => item.id === ticket.id ? { ...item, ...placement } : item))
    try {
      const result = await api.placeTicket(ticket.id, placement)
      setTickets((current) => current.map((item) => item.id === ticket.id ? result.ticket : item))
    } catch (error) {
      setNotice(messageFrom(error))
      try {
        const response = await api.tickets()
        setTickets(response.tickets)
      } catch {
        // The next normal refresh will reconcile the layout.
      }
    }
  }

  function sendToDesk(ticket: Ticket) {
    void updateTicketPlacement(ticket, nextDeskPlacement())
  }

  function dragTrayTicketToDesk(ticket: Ticket, event: ReactDragEvent<HTMLButtonElement>) {
    setTrayDragging(false)
    const bounds = deskRef.current?.getBoundingClientRect()
    if (!bounds || event.clientX <= 0 || event.clientY <= 0) return
    const deskX = Math.max(.08, Math.min(.92, (event.clientX - bounds.left) / bounds.width))
    const deskY = Math.max(.09, Math.min(.9, (event.clientY - bounds.top) / bounds.height))
    void updateTicketPlacement(ticket, { ...nextDeskPlacement(), deskX, deskY })
    setCatalogOpen(false)
    setNotice(`《${ticket.cardName}》已拖到桌面`)
  }

  function removeFromSlot(ticket: Ticket) {
    void updateTicketPlacement(ticket, {
      location: 'desk',
      deskX: ticket.deskX || .5,
      deskY: ticket.deskY || .45,
      rotation: ticket.rotation,
      zIndex: Math.max(1, ...tickets.map((item) => item.zIndex)) + 1,
    })
  }

  function pinToFirstSlot(ticket: Ticket) {
    if (!user?.cardSlotsOwned) {
      setNotice('请先购买固定卡槽，购买后永久开放10个卡位')
      void openShop('card-slots')
      return
    }
    const occupied = new Set(tickets.flatMap((item) => item.slotIndex ? [item.slotIndex] : []))
    const firstFree = Array.from({ length: 10 }, (_, index) => index + 1).find((index) => !occupied.has(index))
    if (!firstFree) {
      setNotice('固定卡槽已经放满10张刮刮乐')
      return
    }
    setActiveTicket(null)
    void updateTicketPlacement(ticket, {
      location: 'slot',
      deskX: ticket.deskX,
      deskY: ticket.deskY,
      rotation: ticket.rotation,
      zIndex: ticket.zIndex,
      slotIndex: firstFree,
    })
    setNotice(`《${ticket.cardName}》已放入固定卡槽 ${firstFree}`)
  }

  async function enqueueRobot(ticket: Ticket) {
    if (ticket.cardCode === 'all-in') {
      setNotice('《放手一博》只能手动刮奖')
      return
    }
    if (ticket.state !== 'purchased') {
      setNotice('机器人只接收还没有刮开的刮刮乐')
      return
    }
    if (!user?.robotOwned) {
      setNotice('请先购买自动刮奖机器人')
      void openShop('robot')
      return
    }
    if (robot && robot.queue.length >= robot.capacity) {
      setNotice('机器人队列已经放满')
      setRobotOpen(true)
      return
    }
    try {
      const result = await api.enqueueRobot(ticket.id)
      setRobot(result.robot)
      setTickets((current) => current.map((item) => item.id === ticket.id ? result.ticket : item))
      setRobotIntakeTicket(ticket)
      if (robotIntakeTimerRef.current !== null) window.clearTimeout(robotIntakeTimerRef.current)
      robotIntakeTimerRef.current = window.setTimeout(() => {
        setRobotIntakeTicket((current) => current?.id === ticket.id ? null : current)
        robotIntakeTimerRef.current = null
      }, 1250)
      setActiveTicket(null)
      setNotice(`机器人正在吞入《${ticket.cardName}》，已加入刮奖队列`)
    } catch (error) {
      setNotice(messageFrom(error))
    }
  }

  function pointInside(element: HTMLElement | null, point: { x: number; y: number }) {
    if (!element) return false
    const bounds = element.getBoundingClientRect()
    return point.x >= bounds.left && point.x <= bounds.right && point.y >= bounds.top && point.y <= bounds.bottom
  }

  function handleTicketDrop(ticket: Ticket, point: { x: number; y: number }, placement: DeskPlacement) {
    const slotIndex = slotRefs.current.findIndex((slot) => pointInside(slot, point))
    if (slotIndex >= 0) {
      if (!user?.cardSlotsOwned) {
        setNotice('固定卡槽尚未购买')
        void updateTicketPlacement(ticket, { ...placement, location: 'desk' })
        return
      }
      void updateTicketPlacement(ticket, { ...placement, location: 'slot', slotIndex: slotIndex + 1 })
      setNotice(`《${ticket.cardName}》已放入固定卡槽 ${slotIndex + 1}`)
      return
    }
    if (pointInside(redeemZoneRef.current, point)) {
      if (ticket.state === 'purchased') {
        setNotice('这张卡还没有刮开，不能兑奖')
        void updateTicketPlacement(ticket, { ...placement, location: 'desk' })
      } else if (!ticket.reward) {
        setNotice('这张卡没有中奖，无法兑奖')
        void updateTicketPlacement(ticket, { ...placement, location: 'desk' })
      } else {
        void redeemTicket(ticket)
      }
      return
    }
    if (pointInside(robotZoneRef.current, point)) {
      void enqueueRobot(ticket)
      return
    }
    if (pointInside(trashZoneRef.current, point)) {
      if (!user?.trashOwned) {
        setNotice('请先购买垃圾桶，才能丢弃桌面刮刮乐')
        void updateTicketPlacement(ticket, { ...placement, location: 'desk' })
        return
      }
      queueDiscard(ticket)
      return
    }
    void updateTicketPlacement(ticket, { ...placement, location: 'desk' })
  }

  function queueDiscard(ticket: Ticket) {
    if (discardingTicketId) return
    if (discardTimerRef.current !== null) window.clearTimeout(discardTimerRef.current)
    if (discardAnimationTimerRef.current !== null) window.clearTimeout(discardAnimationTimerRef.current)
    if (pendingDiscard) setPendingDiscard(null)
    setDiscardingTicketId(ticket.id)
    setNotice(`正在丢弃《${ticket.cardName}》…`)
    void persistDiscard(ticket)
    discardAnimationTimerRef.current = window.setTimeout(() => {
      discardAnimationTimerRef.current = null
      setDiscardingTicketId(null)
      setTickets((current) => current.filter((item) => item.id !== ticket.id))
      setPendingDiscard(ticket)
      setNotice(`《${ticket.cardName}》已放入垃圾桶，5秒内可以撤销`)
      discardTimerRef.current = window.setTimeout(() => {
        discardTimerRef.current = null
        setPendingDiscard((current) => current?.id === ticket.id ? null : current)
      }, 5000)
    }, 950)
  }

  async function persistDiscard(ticket: Ticket) {
    try {
      await api.discardTicket(ticket.id)
    } catch (error) {
      if (discardAnimationTimerRef.current !== null) window.clearTimeout(discardAnimationTimerRef.current)
      if (discardTimerRef.current !== null) window.clearTimeout(discardTimerRef.current)
      discardAnimationTimerRef.current = null
      discardTimerRef.current = null
      setDiscardingTicketId((current) => current === ticket.id ? null : current)
      setTickets((current) => current.some((item) => item.id === ticket.id) ? current : [ticket, ...current])
      setPendingDiscard((current) => current?.id === ticket.id ? null : current)
      setNotice(messageFrom(error))
    }
  }

  async function undoDiscard() {
    if (!pendingDiscard || busy) return
    const ticket = pendingDiscard
    if (discardTimerRef.current !== null) window.clearTimeout(discardTimerRef.current)
    discardTimerRef.current = null
    setBusy(true)
    try {
      const result = await api.restoreDiscardedTicket(ticket.id)
      setTickets((current) => current.some((item) => item.id === result.ticket.id) ? current : [result.ticket, ...current])
      setNotice(`已撤销丢弃《${ticket.cardName}》`)
      setPendingDiscard(null)
    } catch (error) {
      setNotice(messageFrom(error))
    } finally {
      setBusy(false)
    }
  }

  function startFanHold(event: ReactPointerEvent<HTMLButtonElement>) {
    if (!beginFanHold()) return
    event.currentTarget.setPointerCapture(event.pointerId)
  }

  function startFanKeyboard(event: ReactKeyboardEvent<HTMLButtonElement>) {
    if ((event.key !== ' ' && event.key !== 'Enter') || event.repeat) return
    event.preventDefault()
    beginFanHold()
  }

  function beginFanHold() {
    if (fanBusy || busy || fanHoldTimerRef.current !== null) return false
    if (pendingDiscard) {
      setNotice('请先等待或撤销当前垃圾桶操作，再启动风扇')
      return false
    }
    if (!fan?.owned) {
      void openShop('fan')
      return false
    }
    if (!user?.trashOwned) {
      setNotice('请先购买垃圾桶，风扇才能完成清理')
      void openShop('trash')
      return false
    }
    if (deskTickets.length === 0) {
      setNotice('自由桌面上没有可以吹动的刮刮乐')
      return false
    }
    if (!fan.riskAcknowledged && !fanRiskPending) {
      setFanRiskOpen(true)
      return false
    }
    setFanHolding(true)
    fanHoldTimerRef.current = window.setTimeout(() => void runFan(), 650)
    return true
  }

  function stopFanHold() {
    if (fanHoldTimerRef.current !== null) window.clearTimeout(fanHoldTimerRef.current)
    fanHoldTimerRef.current = null
    if (!fanBusy) setFanHolding(false)
  }

  async function runFan() {
    fanHoldTimerRef.current = null
    if (fanBusy) return
    setFanBusy(true)
    setFanHolding(true)
    try {
      const result = await api.blowFan(newIdempotencyKey('fan'), fanRiskPending || Boolean(fan?.riskAcknowledged))
      setUser(result.user)
      setFan(result.fan)
      setRobot(result.robot)
      setFanRiskPending(false)
      const actions = Object.fromEntries(result.event.cards.map((card) => [card.ticket.id, card.action]))
      setFanActions(actions)
      await new Promise((resolve) => window.setTimeout(resolve, Math.max(1250, 2300 - result.fan.level * 80)))
      const affected = new Map(result.event.cards.map((card) => [card.ticket.id, card]))
      setTickets((current) => current.flatMap((ticket) => {
        const card = affected.get(ticket.id)
        if (!card) return [ticket]
        if (card.action === 'discarded') return []
        return [card.ticket]
      }))
      if (activeTicket && affected.has(activeTicket.id)) setActiveTicket(null)
      const discarded = result.event.cards.filter((card) => card.action === 'discarded').length
      const intercepted = result.event.cards.filter((card) => card.action === 'robot' || card.action === 'caught').length
      const safe = result.event.cards.filter((card) => card.action === 'safe').length
      setNotice(`送风完成：清理${discarded}张，机器人保护${intercepted}张，安全落回${safe}张`)
    } catch (error) {
      setNotice(messageFrom(error))
    } finally {
      setFanActions({})
      setFanBusy(false)
      setFanHolding(false)
    }
  }

  if (booting) {
    return <div className="loading-screen"><span className="spinner" />正在打开桌面…</div>
  }

  if (!user) {
    return <AuthScreen onAuthenticated={handleAuthenticated} />
  }

  const firstCard = cards.find((card) => card.code === 'lingqian-ticket')
  const trayTickets = tickets.filter((ticket) => ticket.location === 'tray')
  const deskTickets = tickets.filter((ticket) => ticket.location === 'desk')
  const slotTickets = tickets.filter((ticket) => ticket.location === 'slot')
  const redeemingTicket = redeemingTicketId ? tickets.find((ticket) => ticket.id === redeemingTicketId) : undefined
  const topZ = Math.max(1, ...deskTickets.map((ticket) => ticket.zIndex))

  return (
    <main className="game-shell">
      <header className="topbar">
        <div className="identity">
          <div className="avatar" aria-hidden="true">🐶</div>
          <div><small>{user.username}</small><strong>金币 {coinFormatter.format(user.balance)}</strong></div>
        </div>
        <nav aria-label="主要功能">
          <button type="button" onClick={() => setDailyOpen(true)}>今日任务</button>
          <button type="button" onClick={() => void openShop('luck')} disabled={busy}>商店</button>
          <button type="button" onClick={() => void openLeaderboard()}>排行榜</button>
          <button type="button" onClick={() => void openHistory()}>记录</button>
          <button type="button" onClick={handleLogout} disabled={busy}>退出</button>
        </nav>
      </header>

      {notice && <div className="notice" role="status">{notice}</div>}
      {coinBurst && <div className="coin-flight" key={coinBurst.key} aria-label={`兑奖获得${coinBurst.reward}金币`}>
        {Array.from({ length: 9 }, (_, index) => <i key={index} style={{ '--coin-index': index } as CSSProperties} />)}
      </div>}

      <section className="desk">
        {trayTickets.length > 0 ? (
          <button type="button" className="catalog-dock" onClick={() => setCatalogOpen(true)} aria-label={`打开刮刮乐托盘，${trayTickets.length}张刮刮乐等待放置`}>
            <span className="catalog-stack" aria-hidden="true"><i /><i /><i /></span>
            <strong>刮刮乐托盘</strong>
            <small>{trayTickets.length} 张待放置</small>
          </button>
        ) : (
          <button type="button" className="catalog-shop-entry" onClick={() => setCatalogOpen(true)} aria-label="打开刮刮乐商店">
            <span aria-hidden="true">券</span><strong>刮刮乐商店</strong><small>选购新刮刮乐</small>
          </button>
        )}

        {catalogOpen && <button type="button" className="catalog-scrim" onClick={() => setCatalogOpen(false)} aria-label="关闭刮刮乐商店" />}
        {catalogOpen && <aside className="catalog-panel">
          <button type="button" className="catalog-close" onClick={() => setCatalogOpen(false)} aria-label="关闭刮刮乐商店">×</button>
          <div className="panel-title"><span>刮刮乐商店</span><small>选中即买即玩 · 实物票面预览</small></div>
          <div className="tray-inventory" aria-label="等待放置的刮刮乐">
            <div className="tray-heading"><strong>临时托盘</strong><span>{trayTickets.length} 张</span></div>
            {trayTickets.length === 0 ? <p>暂无留存。新购买的刮刮乐会直接放到桌面。</p> : (
              <div className="tray-stack">
                {trayTickets.slice(0, 8).map((ticket) => (
                  <button type="button" key={ticket.id} draggable onDragStart={() => setTrayDragging(true)} onDragEnd={(event) => dragTrayTicketToDesk(ticket, event)} onClick={() => { sendToDesk(ticket); setCatalogOpen(false) }} disabled={busy}>
                    <span className={`tray-ticket-face card-${ticket.cardCode}`}><img src={unopenedTicketArtworkFor(ticket.cardCode)} alt="" /></span><span>{ticket.cardName}</span><small>{ticket.source === 'daily_wheel' ? '免费刮刮乐' : `${ticket.nominalPrice} 金币`}</small><strong>拖出或点击放置</strong>
                  </button>
                ))}
                {trayTickets.length > 8 && <small>还有 {trayTickets.length - 8} 张等待放置</small>}
              </div>
            )}
          </div>
          {firstCard && (
            <article className="featured-card">
              <span className={`featured-ticket-face card-${firstCard.code}`}><img src={unopenedTicketArtworkFor(firstCard.code)} alt="零钱小票未刮票面" /></span>
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

          <div className="tier-list" aria-label="全部刮刮乐">
            {cards.filter((card) => card.code !== 'lingqian-ticket').map((card, index) => (
              <button className="tier-row" key={card.code} type="button" disabled={!card.implemented || !card.unlocked || busy} onClick={() => purchase(card)}>
                <span className="tier-number">{index + 2}</span>
                <span className={`tier-ticket-face card-${card.code}`}><img src={unopenedTicketArtworkFor(card.code)} alt="" /></span>
                <div><strong>{card.name}</strong><small>{coinFormatter.format(card.price)} 金币门槛</small></div>
                <span className={card.unlocked ? 'unlocked' : 'locked'}>{!card.implemented ? '待开发' : card.unlocked ? '购买' : '未解锁'}</span>
              </button>
            ))}
          </div>
        </aside>}

        <section className="table-area" aria-labelledby="ticket-heading">
          <div className="table-heading">
            <div><span className="eyebrow">MY DESK</span><h1 id="ticket-heading">我的桌面</h1></div>
            <p>未兑奖卡会一直保留</p>
          </div>

          <div className={`desk-zones ${fanHolding ? 'fan-active' : ''} ${redeemingTicket ? 'is-redeeming' : ''}`}>
            <button
              type="button"
              className={`fan-station ${fan?.owned ? 'owned' : 'locked'} ${fanHolding ? 'working' : ''}`}
              onPointerDown={startFanHold}
              onPointerUp={stopFanHold}
              onPointerCancel={stopFanHold}
              onKeyDown={startFanKeyboard}
              onKeyUp={(event) => { if (event.key === ' ' || event.key === 'Enter') stopFanHold() }}
              onContextMenu={(event) => event.preventDefault()}
            >
              <span aria-hidden="true">✺</span><strong>{fan?.owned ? '按住风扇' : '清理风扇'}</strong>
              <small>{fan?.owned ? `${fan.forceText} · 吹错${fan.mistakePercent}%` : '购买 · 500金币'}</small>
              {fanHolding && <i>送风中</i>}
            </button>
            <div className={`redeem-drop-zone ${redeemingTicket ? 'receiving' : ''}`} ref={redeemZoneRef}>
              <span>兑奖区</span><strong>中奖刮刮乐拖到这里</strong><small>未中奖的不会被兑换</small>
              {redeemingTicket && <img className="redeem-intake-ticket" src={ticketArtworkFor(redeemingTicket.cardCode)} alt="" />}
            </div>
            <div className={`trash-drop-zone ${user.trashOwned ? '' : 'locked-zone'}`} ref={trashZoneRef}>
              <span aria-hidden="true">🗑️</span><strong>垃圾桶</strong><small>拖入后可撤销5秒</small>
              {!user.trashOwned && <button type="button" onClick={() => void openShop('trash')} disabled={busy}>购买 · 100金币</button>}
            </div>
          </div>

          <section className={`fixed-slot-panel ${user.cardSlotsOwned ? '' : 'locked-zone'}`} aria-label="固定卡槽">
            <div className="fixed-slot-heading"><div><span className="eyebrow">PROTECTED STORAGE</span><h2>固定卡槽</h2></div><small>{slotTickets.length} / 10</small></div>
            {!user.cardSlotsOwned ? (
              <div className="slot-lock-copy"><span>▥</span><strong>10个保护卡位尚未开放</strong><small>一次购买，永久使用</small><button type="button" className="gold-button" onClick={() => void openShop('card-slots')} disabled={busy}>购买 · 500金币</button></div>
            ) : <div className="fixed-slots">
              {Array.from({ length: 10 }, (_, index) => {
                const slotTicket = slotTickets.find((ticket) => ticket.slotIndex === index + 1)
                const knownProgress = slotTicket ? scratchProgress[slotTicket.id] : undefined
                const visuallyComplete = Boolean(slotTicket && slotTicket.state !== 'purchased' && (knownProgress === undefined || knownProgress >= 100))
                const visualProgress = knownProgress ?? (visuallyComplete ? 100 : 0)
                const fullCoatingPreview = Boolean(slotTicket && slotTicket.state === 'purchased' && usesFullCoatingPreview(slotTicket.cardCode))
                return (
                  <div className={`fixed-slot ${slotTicket ? 'occupied' : ''}`} key={index} ref={(element) => { slotRefs.current[index] = element }} data-slot-index={index + 1}>
                    <span>{index + 1}</span>
                    {slotTicket ? (
                      <div
                        className={`slot-ticket card-${slotTicket.cardCode} state-${slotTicket.state} ${visuallyComplete ? 'visual-complete' : visualProgress > 0 ? 'visual-partial' : 'visual-new'}`}
                        style={{ '--scratch-progress': `${visualProgress}%` } as CSSProperties}
                      >
                        <button type="button" onClick={() => openTicket(slotTicket)}>
                          <img src={fullCoatingPreview ? ticketArtworkFor(slotTicket.cardCode) : slotTicket.state === 'purchased' && visualProgress <= 0 ? unopenedTicketArtworkFor(slotTicket.cardCode) : ticketArtworkFor(slotTicket.cardCode)} alt={`${slotTicket.cardName}票面`} />
                          {slotTicket.state === 'purchased' && (fullCoatingPreview || visualProgress > 0) && <TicketScratchCoating cardCode={slotTicket.cardCode} progress={fullCoatingPreview ? 0 : visualProgress} snapshot={scratchSnapshots[slotTicket.id]} />}
                          <TicketResultSurface ticket={slotTicket} compact />
                          <strong>{slotTicket.cardName}</strong>
                          <small>{!visuallyComplete
                            ? (visualProgress > 0 ? `已刮 ${visualProgress}%` : '未开始')
                            : (slotTicket.reward ? `已刮完 · 中奖${slotTicket.reward}` : '已刮完 · 未中奖')}</small>
                        </button>
                        <button type="button" onClick={() => removeFromSlot(slotTicket)} aria-label={`取出${slotTicket.cardName}`}>取出</button>
                      </div>
                    ) : <small>拖入保护</small>}
                  </div>
                )
              })}
            </div>}
          </section>

          <div className={`free-desk ${fanHolding ? 'fan-active' : ''} ${trayDragging ? 'tray-drop-ready' : ''}`} ref={deskRef} aria-label="可自由摆放刮刮乐的桌面">
            {robotIntakeTicket && (
              <div
                className={`robot-intake-ticket card-${robotIntakeTicket.cardCode}`}
                style={{
                  left: `${robotIntakeTicket.deskX * 100}%`,
                  top: `${robotIntakeTicket.deskY * 100}%`,
                  '--robot-intake-x': `${(0.88 - robotIntakeTicket.deskX) * 100}vw`,
                  '--robot-intake-y': `${(0.55 - robotIntakeTicket.deskY) * 100}vh`,
                  '--robot-intake-x-mid': `${(0.88 - robotIntakeTicket.deskX) * 72}vw`,
                  '--robot-intake-y-mid': `${(0.55 - robotIntakeTicket.deskY) * 72}vh`,
                  '--ticket-rotation': `${robotIntakeTicket.rotation}deg`,
                } as CSSProperties}
                aria-hidden="true"
              >
                <img src={unopenedTicketArtworkFor(robotIntakeTicket.cardCode)} alt="" />
              </div>
            )}
            <button
              type="button"
              ref={robotZoneRef}
              className={`robot-station ${user.robotOwned ? 'owned' : 'locked'} ${robot?.queue.length ? 'processing' : ''} ${robotIntakeTicket ? 'intaking' : ''} ${robotEjectedTicketId ? 'ejecting' : ''}`}
              onClick={() => user.robotOwned ? setRobotOpen(true) : void openShop('robot')}
            >
              <span className={robot?.queue.length ? 'working' : ''}>▣</span>
              <strong>{user.robotOwned ? '自动刮奖机器人' : '机器人维修箱'}</strong>
              <small>{user.robotOwned ? `${robot?.queue.length ?? 0} / ${robot?.capacity ?? 3} 张` : '购买 · 1,000金币'}</small>
              {robot?.queue[0] && <i style={{ width: `${Math.max(3, 100 - robot.queue[0].remainingMs / ((robot.durationSeconds || 1) * 10))}%` }} />}
            </button>
            {robotOutputTicket && (
              <div className={`robot-output-ticket card-${robotOutputTicket.cardCode}`} aria-hidden="true">
                <img src={ticketArtworkFor(robotOutputTicket.cardCode)} alt="" />
                <TicketResultSurface ticket={robotOutputTicket} compact />
              </div>
            )}
            {deskTickets.length === 0 && (
              <div className="empty-free-desk"><span>✦</span><strong>桌面暂无刮刮乐</strong><small>打开商店购买后会直接放到桌面</small></div>
            )}
            {deskTickets.map((ticket) => (
              <DeskTicket
                key={ticket.id}
                ticket={ticket}
                deskRef={deskRef}
                topZ={topZ}
                onOpen={openTicket}
                onPin={pinToFirstSlot}
                onRobot={(ticket) => void enqueueRobot(ticket)}
                fanAction={fanActions[ticket.id]}
                motion={ticket.id === robotEjectedTicketId ? 'robot-ejected' : ticket.id === redeemingTicketId ? 'redeeming' : ticket.id === discardingTicketId ? 'discarding' : undefined}
                scratchProgress={scratchProgress[ticket.id] ?? 0}
                scratchSnapshot={scratchSnapshots[ticket.id]}
                onDrop={handleTicketDrop}
              />
            ))}
          </div>
        </section>
      </section>

      {pendingDiscard && (
        <div className="undo-toast" role="status">
          <span>《{pendingDiscard.cardName}》已放入垃圾桶</span>
          <button type="button" onClick={undoDiscard}>撤销</button>
        </div>
      )}

      {activeTicket && (
        <div className="modal-backdrop" role="presentation" onMouseDown={(event) => {
          if (event.target === event.currentTarget) setActiveTicket(null)
        }}>
          <section className={`scratch-dialog scratch-dialog-${activeTicket.cardCode}`} role="dialog" aria-modal="true" aria-label="刮奖">
            <button className="close-button" type="button" onClick={() => setActiveTicket(null)} aria-label="关闭">×</button>
            <img className="scratch-zoom-artwork" src={ticketArtworkFor(activeTicket.cardCode)} alt={`${activeTicket.cardName}刮刮乐票面`} />
            <div className="scratch-zoom-layer">
            {scratchRequired && !scratchComplete && <TicketPrizeGuide cardCode={activeTicket.cardCode} />}
            {scratchRequired ? (
              <ScratchCard cardCode={activeTicket.cardCode} cardName={activeTicket.cardName} symbols={activeTicket.symbols ?? []} scratchLevel={user.scratchLevel} prizeTier={activeTicket.prizeTier} initialSnapshot={scratchSnapshots[activeTicket.id]} onSnapshot={(snapshot) => rememberScratchSnapshot(activeTicket.id, snapshot)} onProgress={(progress) => rememberScratchProgress(activeTicket.id, progress)} onComplete={() => void completeScratch(activeTicket.id)} />
            ) : (
              <ResultSymbols cardCode={activeTicket.cardCode} cardName={activeTicket.cardName} symbols={activeTicket.symbols ?? []} prizeTier={activeTicket.prizeTier} />
            )}
            {activeTicket.cardCode === 'eternal-color-diamond' && <p className="special-card-rule">五项鉴定中，以最低等级作为本张彩钻的最终等级。</p>}
            {activeTicket.cardCode === 'all-in' && <p className="special-card-rule danger">固定结果规则 · 好运道具无效 · 机器人禁用 · 仅可手动刮开</p>}
            {scratchComplete && (
              <div className={`result-box card-result-${activeTicket.cardCode} tier-${activeTicket.prizeTier ?? 'none'} ${activeTicket.reward ? 'winner' : 'loser'} ${activeTicket.prizeTier === 'jackpot' ? 'jackpot-hit' : ''}`}>
                <small>本张结果</small>
                <h2>{ticketResultTitle(activeTicket)}</h2>
                <p>{activeTicket.reward ? '把中奖卡放入兑奖区即可入账。' : '未中奖卡将继续留在桌面，后续可丢入垃圾桶。'}</p>
                {activeTicket.state === 'scratched' && Boolean(activeTicket.reward) && (
                  <button type="button" className="gold-button" onClick={redeem} disabled={busy}>手动兑奖</button>
                )}
                {activeTicket.state !== 'redeemed' && activeTicket.location === 'slot' ? (
                  <button type="button" className="secondary-button" onClick={() => { removeFromSlot(activeTicket); setActiveTicket(null) }} disabled={busy}>从固定卡槽取出</button>
                ) : activeTicket.state !== 'redeemed' && (
                  <button type="button" className="secondary-button" onClick={() => pinToFirstSlot(activeTicket)} disabled={busy || slotTickets.length >= 10}>放入固定卡槽</button>
                )}
                {activeTicket.state === 'redeemed' && <span className="redeemed-badge">已经兑奖</span>}
              </div>
            )}
            </div>
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

      {shopOpen && shop && user && (
        <ShopDialog user={user} shop={shop} busy={busy} initialItemCode={shopFocus} onClose={() => setShopOpen(false)} onUpgrade={upgradeItem} />
      )}

      {robotOpen && robot?.owned && (
        <RobotDialog robot={robot} onClose={() => setRobotOpen(false)} onOpenShop={(code) => { setRobotOpen(false); void openShop(code) }} />
      )}

      {fanRiskOpen && (
        <div className="modal-backdrop" role="presentation">
          <section className="fan-risk-dialog" role="dialog" aria-modal="true" aria-label="风扇风险说明">
            <span className="fan-risk-icon" aria-hidden="true">✺</span>
            <h2>启动风扇前请确认</h2>
            <p>风扇会吹动自由桌面上的所有刮刮乐。已刮完的会直接进入垃圾桶；未刮完的如果没有被机器人拦截，也可能因吹错而被丢弃。</p>
            <p>固定卡槽内的卡完全不受影响。风扇造成的丢弃不会退还金币，也不会补发奖励。</p>
            <div><button type="button" className="secondary-button" onClick={() => setFanRiskOpen(false)}>暂不使用</button><button type="button" className="gold-button" onClick={() => { setFanRiskPending(true); setFanRiskOpen(false); setNotice('风险已确认，请持续按住风扇启动') }}>我已了解</button></div>
          </section>
        </div>
      )}

      {leaderboardOpen && (
        <LeaderboardDialog leaderboard={leaderboard} loading={leaderboardLoading} error={leaderboardError} onClose={() => setLeaderboardOpen(false)} onRefresh={() => void openLeaderboard()} />
      )}

      {historyOpen && (
        <HistoryDialog events={historyEvents} loading={historyLoading} error={historyError} onClose={() => setHistoryOpen(false)} onRefresh={() => void openHistory()} />
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
        <span className="eyebrow">SCRATCH CARD GAME</span>
        <h1>欢迎来到<br />刮刮乐小游戏</h1>
        <p>注册账号即可开始游戏。游戏仅使用虚拟金币，不支持充值、提现或交易。</p>
        <ul>
          <li>新账号获得 1,000 虚拟金币</li>
          <li>完成每日任务可获得金币和免费卡</li>
          <li>积累金币，逐步解锁更多主题刮刮卡</li>
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
          <span className="eyebrow">DAILY TASKS</span>
          <h1 id="daily-title">今日任务</h1>
          <p>{daily.date} · 每日零点刷新，未使用次数不累计</p>
        </header>

        <div className="daily-grid">
          <article className={`task-card ${daily.loginClaimed ? 'task-complete' : ''}`}>
            <div className="task-icon" aria-hidden="true">☀️</div>
            <div className="task-copy">
              <span>每日登录</span>
              <h2>领取 100 金币</h2>
              <p>每天登录一次即可领取。</p>
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
              <div className="wheel-result"><span>今日结果</span><strong>《{daily.wheelCardName ?? '免费刮刮乐'}》</strong><small>刮刮乐已经放到桌面</small></div>
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

function ResultSymbols({ cardCode, cardName, symbols, prizeTier = 'none' }: { cardCode: string; cardName: string; symbols: string[]; prizeTier?: string }) {
  const tripleMatch = symbols.length === 3 && new Set(symbols).size === 1
  const { stageHeight, cells } = scratchLayoutFor(cardCode, symbols.length)
  return (
    <div className={`result-ticket result-ticket-${cardCode} tier-${prizeTier} ${tripleMatch ? 'is-triple' : ''}`}>
      <span>{cardName}</span>
      <div className={`result-symbols authored-result-layout count-${symbols.length}`}>{symbols.map((symbol, index) => {
        const visual = getSymbolVisual(symbol)
        const cell = cells[index]
        return <strong
          className={`symbol-${symbolClassName(symbol)} ${symbolStateClassNames(cardCode, symbols, index)}`}
          aria-label={visual.label}
          key={`${symbol}-${index}`}
          style={{ left: `${cell.x / 7.2}%`, top: `${cell.y / stageHeight * 100}%`, width: `${cell.width / 7.2}%`, height: `${cell.height / stageHeight * 100}%` }}
        ><TicketSymbolGlyph cardCode={cardCode} symbol={symbol} /><small>{symbol.replace('目标·', '目标：')}</small></strong>
      })}</div>
    </div>
  )
}

const ticketPrizeGuides: Record<string, { label: string; value: string; jackpot?: boolean }[]> = {
  'lingqian-ticket': [
    { label: '狗头金币', value: '25 / 三同50' }, { label: '钞票', value: '50 / 三同100' },
    { label: '碎钻石', value: '60 / 三同120' }, { label: '钞票堆', value: '75 / 头奖150', jackpot: true },
  ],
  'street-store': [
    { label: '辣条', value: '750' }, { label: '可乐', value: '1,500' }, { label: '冰棍', value: '1,800' },
    { label: '雪糕', value: '2,250' }, { label: '玩具车', value: '3,000' }, { label: '小电视', value: '3,750' },
    { label: '游戏机', value: '头奖 4,500', jackpot: true },
  ],
  'arcade-challenge': [
    { label: '弹珠', value: '2,500' }, { label: '拳套', value: '5,000' }, { label: '赛车', value: '6,500' },
    { label: '飞机', value: '10,000' }, { label: '皇冠', value: '头奖 15,000', jackpot: true },
  ],
  'gold-mine': [
    { label: '2个', value: '7,500' }, { label: '3个', value: '15,000' }, { label: '4个', value: '22,500' },
    { label: '5个', value: '30,000' }, { label: '6个', value: '头奖 45,000', jackpot: true },
  ],
  'rocket-launch': [
    { label: '4–5', value: '20,000' }, { label: '6–7', value: '40,000' }, { label: '8', value: '60,000' },
    { label: '9–10', value: '80,000' }, { label: '11–12', value: '头奖 120,000', jackpot: true },
  ],
  'deep-sea-salvage': [
    { label: '海草团', value: '40,000' }, { label: '破皮靴', value: '80,000' }, { label: '漂流瓶', value: '96,000' },
    { label: '船锚', value: '120,000' }, { label: '珍珠贝', value: '160,000' }, { label: '宝箱', value: '200,000' },
    { label: '王冠', value: '头奖 240,000', jackpot: true },
  ],
  'eternal-color-diamond': [
    { label: '裂纹', value: '75,000' }, { label: '工业', value: '150,000' }, { label: '珠宝', value: '180,000' },
    { label: '稀有', value: '225,000' }, { label: '精选', value: '240,000' }, { label: '典藏', value: '300,000' },
    { label: '皇室', value: '375,000' }, { label: '永恒', value: '头奖 450,000', jackpot: true },
  ],
}

function TicketPrizeGuide({ cardCode }: { cardCode: string }) {
  const prizes = ticketPrizeGuides[cardCode]
  if (!prizes) return null
  return (
    <div className={`ticket-prize-guide guide-${cardCode} count-${prizes.length}`} aria-label="本卡奖级">
      {prizes.map((prize) => <span className={prize.jackpot ? 'jackpot' : ''} key={prize.label}>
        <small>{prize.label}</small><strong>{prize.value}</strong>
      </span>)}
    </div>
  )
}

function ticketResultTitle(ticket: Ticket) {
  const jackpotPrefix = ticket.prizeTier === 'jackpot' ? '🎉 头奖 · ' : ''
  if (ticket.cardCode === 'eternal-color-diamond') {
    const grades = ['裂纹级', '工业级', '珠宝级', '稀有级', '精选级', '典藏级', '皇室级', '永恒级']
    const finalGrade = (ticket.symbols ?? []).reduce((lowest, grade) => grades.indexOf(grade) < grades.indexOf(lowest) ? grade : lowest, '永恒级')
    return `${jackpotPrefix}最终鉴定 ${finalGrade} · 获得 ${coinFormatter.format(ticket.reward ?? 0)} 金币`
  }
  if (ticket.cardCode === 'all-in') {
    const symbol = ticket.symbols?.[0] ?? '骷髅头'
    return ticket.reward ? `${symbol} · 获得 ${coinFormatter.format(ticket.reward)} 金币` : `${symbol} · 未中奖`
  }
  return ticket.reward ? `${jackpotPrefix}获得 ${coinFormatter.format(ticket.reward)} 金币` : '未中奖'
}

function messageFrom(error: unknown) {
  return error instanceof Error ? error.message : '发生未知错误，请稍后重试'
}
