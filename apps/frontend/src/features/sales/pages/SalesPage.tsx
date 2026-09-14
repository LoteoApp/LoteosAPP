import { useEffect, useState } from 'react'
import { messageFromError } from '../../../shared/api/client'
import LoteCombobox from '../components/LoteCombobox'
import SaleForm from '../components/SaleForm'
import type { ClienteOption, LoteOption, NewClientValues, SellerOption } from '../types'

export type SalesPageProps = {
  // The composition root injects every data source: sales never imports
  // another feature's files.
  loadLotes: (signal?: AbortSignal) => Promise<LoteOption[]>
  loadClientes: (signal?: AbortSignal) => Promise<ClienteOption[]>
  createCliente: (values: NewClientValues) => Promise<ClienteOption>
  // Eligibility depends on the loteo of the lote being sold, so the sellers
  // can only be loaded once there is a lote.
  loadSellers: (loteoId: string, signal?: AbortSignal) => Promise<SellerOption[]>
}

export default function SalesPage({
  loadLotes,
  loadClientes,
  createCliente,
  loadSellers,
}: SalesPageProps) {
  const [lotes, setLotes] = useState<LoteOption[]>([])
  const [lote, setLote] = useState<LoteOption | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)

  useEffect(() => {
    const controller = new AbortController()

    async function load() {
      setIsLoading(true)
      try {
        const loaded = await loadLotes(controller.signal)
        if (controller.signal.aborted) {
          return
        }
        setLotes(loaded)
        setLoadError(null)
      } catch (error) {
        if (controller.signal.aborted) {
          return
        }
        setLoadError(messageFromError(error))
      } finally {
        if (!controller.signal.aborted) {
          setIsLoading(false)
        }
      }
    }

    void load()

    return () => controller.abort()
  }, [loadLotes])

  return (
    <section className="flex flex-col gap-4">
      <div className="flex flex-col gap-2">
        <h1 className="text-2xl font-semibold text-foreground">Nueva venta</h1>
        <p className="text-muted-foreground">
          Elegí el lote, el cliente comprador y quién realizó la venta.
        </p>
      </div>

      {loadError !== null && (
        <p role="alert" className="text-sm text-destructive">
          {loadError}
        </p>
      )}

      <SaleForm
        lote={lote}
        loadClientes={loadClientes}
        createCliente={createCliente}
        loadSellers={loadSellers}
        disabled={isLoading}
        leading={
          <LoteCombobox
            lotes={lotes}
            value={lote}
            onChange={setLote}
            isLoading={isLoading}
          />
        }
      />
    </section>
  )
}
