import { useEffect, useMemo, useState } from 'react'
import type { ShopItem, ShopStatus, User } from './api'

const coins = new Intl.NumberFormat('zh-CN')

type Props = {
  user: User
  shop: ShopStatus
  busy: boolean
  initialItemCode?: ShopItem['code']
  onClose: () => void
  onUpgrade: (item: ShopItem) => void
}

const itemMeta: Record<ShopItem['code'], { icon: string; group: string }> = {
  luck: { icon: '✦', group: '长期成长' },
  'scratch-range': { icon: '⌁', group: '操作效率' },
  trash: { icon: '🗑', group: '桌面整理' },
  'card-slots': { icon: '▥', group: '卡片保护' },
  robot: { icon: '▣', group: '自动刮奖' },
  'robot-speed': { icon: '⚡', group: '处理速度' },
  'robot-queue': { icon: '☷', group: '等待容量' },
  'robot-intercept': { icon: '⬡', group: '卡片保护' },
}

function percent(basisPoint: number) {
  return `${(basisPoint / 100).toFixed(2)}%`
}

export function ShopDialog({ user, shop, busy, initialItemCode, onClose, onUpgrade }: Props) {
  const fallbackCode = shop.items[0]?.code ?? 'luck'
  const [selectedCode, setSelectedCode] = useState<ShopItem['code']>(initialItemCode ?? fallbackCode)
  const [confirming, setConfirming] = useState(false)
  const [showOdds, setShowOdds] = useState(false)
  const selected = shop.items.find((item) => item.code === selectedCode) ?? shop.items[0]
  const category = selected?.category ?? 'luck'
  const visibleItems = useMemo(() => shop.items.filter((item) => item.category === category), [category, shop.items])

  useEffect(() => {
    if (initialItemCode) setSelectedCode(initialItemCode)
  }, [initialItemCode])
  useEffect(() => {
    setConfirming(false)
    setShowOdds(false)
  }, [selectedCode, selected?.level])
  if (!selected) return null

  const maxed = selected.level >= selected.maxLevel
  const nextPrice = selected.nextPrice ?? 0
  const affordable = user.balance >= nextPrice
  const afterBalance = user.balance - nextPrice
  const oneTime = selected.maxLevel === 1
  const action = oneTime ? '购买' : '升级'

  function selectCategory(nextCategory: ShopItem['category']) {
    const first = shop.items.find((item) => item.category === nextCategory)
    if (first) setSelectedCode(first.code)
  }

  return (
    <div className="modal-backdrop shop-backdrop" role="presentation" onMouseDown={(event) => {
      if (event.target === event.currentTarget && !busy) onClose()
    }}>
      <section className="shop-dialog" role="dialog" aria-modal="true" aria-labelledby="shop-title">
        <header className="shop-header">
          <div><span className="eyebrow">ITEM SHOP</span><h1 id="shop-title">道具商店</h1></div>
          <div className="shop-balance"><small>当前余额</small><strong>金币 {coins.format(user.balance)}</strong></div>
          <button type="button" className="close-button" onClick={onClose} disabled={busy} aria-label="关闭商店">×</button>
        </header>

        <nav className="shop-tabs" aria-label="商店分类">
          <button type="button" className={category === 'luck' ? 'active' : ''} onClick={() => selectCategory('luck')}>好运</button>
          <button type="button" className={category === 'efficiency' ? 'active' : ''} onClick={() => selectCategory('efficiency')}>效率</button>
          <button type="button" className={category === 'safety' ? 'active' : ''} onClick={() => selectCategory('safety')}>整理与安全</button>
        </nav>

        <div className="shop-layout">
          <div className="shop-items">
            {visibleItems.map((item) => (
              <button type="button" key={item.code} className={`shop-item ${selected.code === item.code ? 'selected' : ''}`} onClick={() => setSelectedCode(item.code)}>
                <span className={`shop-item-icon ${item.code}`}>{itemMeta[item.code].icon}</span>
                <div><small>{itemMeta[item.code].group}</small><h2>{item.name}</h2><p>{item.locked ? item.lockedReason : item.maxLevel === 1 ? item.effectText : `等级 ${item.level} / ${item.maxLevel}`}</p></div>
                <strong>{item.locked ? '需要前置' : item.level >= item.maxLevel ? (item.maxLevel === 1 ? '已拥有' : '已满级') : `${coins.format(item.nextPrice ?? 0)} 金币`}</strong>
              </button>
            ))}
          </div>

          <article className="shop-detail">
            <div className="shop-detail-title"><span className={`shop-item-icon ${selected.code}`}>{itemMeta[selected.code].icon}</span><div><small>永久道具</small><h2>{selected.name}</h2></div></div>
            <p>{selected.description}</p>
            {selected.notice && <div className="shop-note">{selected.notice}</div>}
            <div className="level-progress"><span style={{ width: `${(selected.level / selected.maxLevel) * 100}%` }} /></div>
            <div className="effect-change">
              <div><small>当前效果</small><strong>{selected.effectText}</strong></div>
              <span>→</span>
              <div><small>{maxed ? '当前状态' : oneTime ? '购买后' : `升级至 ${selected.level + 1} 级`}</small><strong>{maxed ? selected.effectText : selected.nextEffectText}</strong></div>
            </div>
            <dl className="shop-costs">
              <div><dt>累计投入</dt><dd>{coins.format(selected.totalSpent)} 金币</dd></div>
              {!maxed && !selected.locked && <div><dt>本次{action}</dt><dd>{coins.format(nextPrice)} 金币</dd></div>}
              {!maxed && !selected.locked && <div><dt>{action}后余额</dt><dd>{coins.format(Math.max(0, afterBalance))} 金币</dd></div>}
            </dl>
            {!maxed && selected.relockedCards.length > 0 && <div className="relock-warning">{action}后将暂时重新锁定：{selected.relockedCards.join('、')}</div>}

            {selected.code === 'luck' && (
              <div className="luck-impact">
                <button type="button" className="odds-toggle" onClick={() => setShowOdds((value) => !value)}>{showOdds ? '收起真实概率' : '查看当前真实概率与返奖率'}</button>
                {showOdds && <div className="luck-cards">
                  {shop.luckCards.map((card) => (
                    <details key={card.cardCode}>
                      <summary><strong>{card.cardName}</strong><span>理论返奖率 {percent(card.currentRtpBasisPoint)}{card.currentLevel !== card.nextLevel && ` → ${percent(card.nextRtpBasisPoint)}`}</span></summary>
                      <div className="luck-tier-table">
                        <div><strong>结果</strong><strong>当前 L{card.currentLevel}</strong>{card.currentLevel !== card.nextLevel && <strong>下级 L{card.nextLevel}</strong>}</div>
                        {card.tiers.map((tier) => <div key={tier.label}><span>{tier.label}<small>{tier.rewardText}</small></span><span>{percent(tier.currentBasisPoint)}</span>{card.currentLevel !== card.nextLevel && <span>{percent(tier.nextBasisPoint)}</span>}</div>)}
                      </div>
                    </details>
                  ))}
                  <p className="rtp-note">理论返奖率是大量重复游戏的统计平均值，不代表单张卡一定获得对应回报。</p>
                </div>}
              </div>
            )}

            {selected.locked ? <div className="locked-item-badge">{selected.lockedReason}</div> : maxed ? <div className="max-level-badge">{oneTime ? '已永久拥有' : '已达到最高等级'}</div> : confirming ? (
              <div className="upgrade-confirm">
                <p>确认花费 <strong>{coins.format(nextPrice)} 金币</strong>{oneTime ? `购买“${selected.name}”` : `将“${selected.name}”升级到 ${selected.level + 1} 级`}？</p>
                <div><button type="button" className="secondary-button" onClick={() => setConfirming(false)} disabled={busy}>取消</button><button type="button" className="gold-button" onClick={() => onUpgrade(selected)} disabled={busy}>{busy ? `${action}中…` : `确认${action}`}</button></div>
              </div>
            ) : (
              <button type="button" className="gold-button shop-upgrade" onClick={() => setConfirming(true)} disabled={!affordable || busy}>{affordable ? (oneTime ? `购买 · ${coins.format(nextPrice)} 金币` : `升级到 ${selected.level + 1} 级`) : `还差 ${coins.format(nextPrice - user.balance)} 金币`}</button>
            )}
          </article>
        </div>
      </section>
    </div>
  )
}
