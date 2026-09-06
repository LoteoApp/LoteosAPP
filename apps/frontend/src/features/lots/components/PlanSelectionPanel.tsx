import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '../../../shared/ui/card'
import { formatArea } from '../../../shared/lib/formatArea'
import { formatCurrency } from '../../../shared/lib/formatCurrency'
import type { UpdateCallePayload } from '../api/update-calle'
import type { UpdateManzanaPayload } from '../api/update-manzana'
import type { UpdateCalleState } from '../hooks/use-update-calle'
import type { UpdateLoteState } from '../hooks/use-update-lote'
import type { UpdateManzanaState } from '../hooks/use-update-manzana'
import type { UpdateLotePayload } from '../lib/loteFormValues'
import type { LoteoDetail, PlanEntityRef } from '../types'
import CalleEditForm from './CalleEditForm'
import LoteEditForm from './LoteEditForm'
import ManzanaEditForm from './ManzanaEditForm'

const EMPTY_PROMPT = 'Tocá una manzana, un lote o una calle.'

type PlanSelectionPanelProps = {
  canEdit?: boolean
  selected: PlanEntityRef | null
  loteo: LoteoDetail
  polygonLabels: ReadonlyMap<string, string>
  selectedPolygonId: string | null
  updateState: UpdateLoteState
  onSave: (loteId: string, payload: UpdateLotePayload) => Promise<boolean>
  manzanaUpdateState: UpdateManzanaState
  onSaveManzana: (manzanaId: string, payload: UpdateManzanaPayload) => Promise<boolean>
  calleUpdateState: UpdateCalleState
  onSaveCalle: (calleId: string, payload: UpdateCallePayload) => Promise<boolean>
}

export default function PlanSelectionPanel({
  canEdit = true,
  selected,
  loteo,
  polygonLabels,
  selectedPolygonId,
  updateState,
  onSave,
  manzanaUpdateState,
  onSaveManzana,
  calleUpdateState,
  onSaveCalle,
}: PlanSelectionPanelProps) {
  const title = titleFor(selected, loteo, polygonLabels, selectedPolygonId)

  if (selected === null || selected.kind === 'loteo') {
    return (
      <Card size="sm">
        <CardHeader>
          <CardTitle>Selección</CardTitle>
          <CardDescription>{EMPTY_PROMPT}</CardDescription>
        </CardHeader>
      </Card>
    )
  }

  if (selected.kind === 'lote') {
    const lote = loteo.lotes.find((item) => item.id === selected.id)
    if (!lote) {
      return (
        <Card size="sm">
          <CardHeader>
            <CardTitle>Selección</CardTitle>
            <CardDescription>{EMPTY_PROMPT}</CardDescription>
          </CardHeader>
        </Card>
      )
    }

    return (
      <Card size="sm">
        <CardHeader>
          <CardTitle>{title}</CardTitle>
          <CardDescription>Número, precio, superficie y características.</CardDescription>
        </CardHeader>
        <CardContent>
          {canEdit ? (
            <LoteEditForm
              lote={lote}
              updateState={updateState}
              onSave={(payload) => onSave(lote.id, payload)}
            />
          ) : (
            <LoteReadOnly lote={lote} />
          )}
        </CardContent>
      </Card>
    )
  }

  if (selected.kind === 'manzana') {
    const manzana = loteo.manzanas.find((item) => item.id === selected.id)
    if (!manzana) {
      return (
        <Card size="sm">
          <CardHeader>
            <CardTitle>Selección</CardTitle>
            <CardDescription>{EMPTY_PROMPT}</CardDescription>
          </CardHeader>
        </Card>
      )
    }

    return (
      <Card size="sm">
        <CardHeader>
          <CardTitle>{title}</CardTitle>
          <CardDescription>Número, servicios y calles de la manzana.</CardDescription>
        </CardHeader>
        <CardContent>
          {canEdit ? (
            <ManzanaEditForm
              manzana={manzana}
              calles={loteo.calles}
              loteCount={loteCountOf(loteo, manzana.id)}
              updateState={manzanaUpdateState}
              onSave={(payload) => onSaveManzana(manzana.id, payload)}
            />
          ) : (
            <ManzanaReadOnly
              manzana={manzana}
              calles={loteo.calles}
              loteCount={loteCountOf(loteo, manzana.id)}
            />
          )}
        </CardContent>
      </Card>
    )
  }

  const calle = loteo.calles.find((item) => item.id === selected.id)
  if (!calle) {
    return (
      <Card size="sm">
        <CardHeader>
          <CardTitle>Selección</CardTitle>
          <CardDescription>{EMPTY_PROMPT}</CardDescription>
        </CardHeader>
      </Card>
    )
  }

  return (
    <Card size="sm">
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        <CardDescription>Nombre y tipo de la calle.</CardDescription>
      </CardHeader>
      <CardContent>
        {canEdit ? (
          <CalleEditForm
            calle={calle}
            updateState={calleUpdateState}
            onSave={(payload) => onSaveCalle(calle.id, payload)}
          />
        ) : (
          <CalleReadOnly calle={calle} />
        )}
      </CardContent>
    </Card>
  )
}

function ReadOnlyRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col gap-0.5 border-b border-border/60 pb-2 last:border-b-0 last:pb-0">
      <dt className="text-xs font-medium text-muted-foreground">{label}</dt>
      <dd className="text-sm">{value || '—'}</dd>
    </div>
  )
}

function LoteReadOnly({ lote }: { lote: LoteoDetail['lotes'][number] }) {
  return (
    <dl className="grid gap-3 sm:grid-cols-2">
      <ReadOnlyRow label="Número" value={lote.numero} />
      <ReadOnlyRow
        label="Precio"
        value={lote.precio === null ? '—' : formatCurrency(lote.precio, lote.moneda)}
      />
      <ReadOnlyRow
        label="Superficie"
        value={lote.superficie === null ? '—' : formatArea(lote.superficie)}
      />
      <ReadOnlyRow label="Características" value={lote.caracteristicas} />
    </dl>
  )
}

function ManzanaReadOnly({
  manzana,
  calles,
  loteCount,
}: {
  manzana: LoteoDetail['manzanas'][number]
  calles: LoteoDetail['calles']
  loteCount: number
}) {
  const services = [
    manzana.tieneAgua ? 'Agua' : null,
    manzana.tieneCloaca ? 'Cloaca' : null,
    manzana.tieneLuz ? 'Luz' : null,
    manzana.tieneGas ? 'Gas' : null,
  ].filter((service): service is string => service !== null)
  const calleNames = manzana.calleIds
    .map((id) => calles.find((calle) => calle.id === id)?.nombre)
    .filter((name): name is string => Boolean(name))

  return (
    <dl className="grid gap-3 sm:grid-cols-2">
      <ReadOnlyRow label="Número" value={manzana.numero} />
      <ReadOnlyRow label="Lotes" value={String(loteCount)} />
      <ReadOnlyRow
        label="Servicios"
        value={services.length > 0 ? services.join(', ') : 'Ninguno'}
      />
      <ReadOnlyRow
        label="Calles"
        value={calleNames.length > 0 ? calleNames.join(', ') : 'Ninguna'}
      />
    </dl>
  )
}

function CalleReadOnly({ calle }: { calle: LoteoDetail['calles'][number] }) {
  return (
    <dl className="grid gap-3 sm:grid-cols-2">
      <ReadOnlyRow label="Nombre" value={calle.nombre} />
      <ReadOnlyRow label="Tipo" value={calle.tipo} />
    </dl>
  )
}

function loteCountOf(loteo: LoteoDetail, manzanaId: string): number {
  return loteo.lotes.filter((lote) => lote.manzanaId === manzanaId).length
}

function titleFor(
  selected: PlanEntityRef | null,
  loteo: LoteoDetail,
  polygonLabels: ReadonlyMap<string, string>,
  selectedPolygonId: string | null,
): string {
  if (selectedPolygonId) {
    const label = polygonLabels.get(selectedPolygonId)
    if (label) {
      return label
    }
  }
  if (selected?.kind === 'lote') {
    const lote = loteo.lotes.find((item) => item.id === selected.id)
    return lote?.numero ? `Lote ${lote.numero}` : 'Lote'
  }
  if (selected?.kind === 'manzana') {
    const manzana = loteo.manzanas.find((item) => item.id === selected.id)
    return manzana?.numero ? `Manzana ${manzana.numero}` : 'Manzana'
  }
  if (selected?.kind === 'calle') {
    const calle = loteo.calles.find((item) => item.id === selected.id)
    return calle?.nombre ? `Calle ${calle.nombre}` : 'Calle'
  }
  return 'Selección'
}
