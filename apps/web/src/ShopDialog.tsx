import { useEffect, useState } from 'react'
import type { ShopItem, ShopStatus, User } from './api'

const coins = new Intl.NumberFormat('zh-CN')

type Props = {
  user: User
  shop: ShopStatus
  busy: boolean
  onClose: () => void
  onUpgrade: (item: ShopItem) => void
}

export function ShopDialog({ user, shop, busy, onClose, onUpgrade }: Props) {
  const [selectedCode, setSelectedCode] = useState(shop.items[0]?.code ?? 'luck')
  const [confirming, setConfirming] = useState(false)
  const selected = shop.items.find((item) => item.code === selectedCode) ?? shop.items[0]

  useEffect(() => setConfirming(false), [selectedCode, selected?.level])
  if (!selected) return null

  const maxed = selected.level >= selected.maxLevel
  const nextPrice = selected.nextPrice ?? 0
  const affordable = user.balance >= nextPrice
  const afterBalance = user.balance - nextPrice
  const effectLabel = selected.code === 'luck' ? '好运系数' : '划动覆盖范围'

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
          <button type="button" className={selected.category === 'luck' ? 'active' : ''} onClick={() => setSelectedCode('luck')}>好运</button>
          <button type="button" className={selected.category === 'efficiency' ? 'active' : ''} onClick={() => setSelectedCode('scratch-range')}>效率</button>
          <button type="button" disabled>整理与安全 · 后续开放</button>
        </nav>

        <div className="shop-layout">
          <div className="shop-items">
            {shop.items.map((item) => (
              <button type="button" key={item.code} className={`shop-item ${selected.code === item.code ? 'selected' : ''}`} onClick={() => setSelectedCode(item.code)}>
                <span className={`shop-item-icon ${item.code}`}>{item.code === 'luck' ? '✦' : '⌁'}</span>
                <div><small>{item.category === 'luck' ? '长期成长' : '操作效率'}</small><h2>{item.name}</h2><p>等级 {item.level} / {item.maxLevel}</p></div>
                <strong>{item.level >= item.maxLevel ? '已满级' : `${coins.format(item.nextPrice ?? 0)} 金币`}</strong>
              </button>
            ))}
          </div>

          <article className="shop-detail">
            <div className="shop-detail-title"><span className={`shop-item-icon ${selected.code}`}>{selected.code === 'luck' ? '✦' : '⌁'}</span><div><small>永久道具</small><h2>{selected.name}</h2></div></div>
            <p>{selected.description}</p>
            {selected.notice && <div className="shop-note">{selected.notice}</div>}
            <div className="level-progress"><span style={{ width: `${(selected.level / selected.maxLevel) * 100}%` }} /></div>
            <div className="effect-change">
              <div><small>当前{effectLabel}</small><strong>{selected.effectPercent}%</strong></div>
              <span>→</span>
              <div><small>{maxed ? '当前状态' : `升级至 ${selected.level + 1} 级`}</small><strong>{maxed ? '已满级' : `${selected.nextEffectPercent}%`}</strong></div>
            </div>
            <dl className="shop-costs">
              <div><dt>累计投入</dt><dd>{coins.format(selected.totalSpent)} 金币</dd></div>
              {!maxed && <div><dt>本次升级</dt><dd>{coins.format(nextPrice)} 金币</dd></div>}
              {!maxed && <div><dt>升级后余额</dt><dd>{coins.format(Math.max(0, afterBalance))} 金币</dd></div>}
            </dl>
            {!maxed && selected.relockedCards.length > 0 && (
              <div className="relock-warning">升级后将暂时重新锁定：{selected.relockedCards.join('、')}</div>
            )}

            {maxed ? <div className="max-level-badge">已达到最高等级</div> : confirming ? (
              <div className="upgrade-confirm">
                <p>确认花费 <strong>{coins.format(nextPrice)} 金币</strong>，将“{selected.name}”升级到 {selected.level + 1} 级？</p>
                <div><button type="button" className="secondary-button" onClick={() => setConfirming(false)} disabled={busy}>取消</button><button type="button" className="gold-button" onClick={() => onUpgrade(selected)} disabled={busy}>{busy ? '升级中…' : '确认升级'}</button></div>
              </div>
            ) : (
              <button type="button" className="gold-button shop-upgrade" onClick={() => setConfirming(true)} disabled={!affordable || busy}>{affordable ? `升级到 ${selected.level + 1} 级` : `还差 ${coins.format(nextPrice - user.balance)} 金币`}</button>
            )}
          </article>
        </div>
      </section>
    </div>
  )
}
