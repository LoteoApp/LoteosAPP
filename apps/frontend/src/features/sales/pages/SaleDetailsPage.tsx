import { useEffect, useState, type ReactNode } from 'react'
import { Printer } from 'lucide-react'
import { Link, useParams } from 'react-router'
import { Alert, AlertDescription, AlertTitle } from '../../../shared/ui/alert'
import { Button, buttonVariants } from '../../../shared/ui/button'
import { getSale } from '../api/sales'
import SaleDetails from '../components/SaleDetails'
import SaleReceiptDialog from '../components/SaleReceiptDialog'
import SalesListSkeleton from '../components/SalesListSkeleton'
import { saleReceiptFromSale, type Sale } from '../types'

type SaleDetailsPageProps = {
  accessToken?: string
  renderPlan?: (sale: Sale) => ReactNode
}

export default function SaleDetailsPage({ accessToken = '', renderPlan }: SaleDetailsPageProps) {
  const token = accessToken
  const { id = '' } = useParams()
  const [sale, setSale] = useState<Sale | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [isReceiptOpen, setIsReceiptOpen] = useState(false)
  const queryEnabled = token !== ''
  const requestKey = JSON.stringify([id, token])
  const [loadedKey, setLoadedKey] = useState(requestKey)
  if (requestKey !== loadedKey) {
    setLoadedKey(requestKey)
    setSale(null)
    setIsLoading(queryEnabled)
    setError(null)
    setIsReceiptOpen(false)
  }

  useEffect(() => {
    if (!queryEnabled) return
    const controller = new AbortController()
    getSale(token, id, controller.signal)
      .then((loaded) => {
        if (!controller.signal.aborted) {
          setSale(loaded)
          setError(null)
        }
      })
      .catch((loadError: unknown) => {
        if (!controller.signal.aborted) {
          setError(loadError instanceof Error ? loadError.message : 'No se pudo cargar la venta.')
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) setIsLoading(false)
      })
    return () => controller.abort()
  }, [id, queryEnabled, token])

  return (
    <section className="flex flex-col gap-4">
      <Link to="/ventas" className="w-fit text-sm text-muted-foreground hover:text-foreground">
        Volver a ventas
      </Link>
      {queryEnabled && isLoading && <SalesListSkeleton rows={1} />}
      {queryEnabled && error && (
        <Alert variant="destructive">
          <AlertTitle>No se pudo cargar la venta</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}
      {queryEnabled && !isLoading && !error && sale && (
        <div className="grid min-w-0 gap-4 lg:grid-cols-12 lg:items-start">
          <div className="min-w-0 lg:col-span-5">{renderPlan?.(sale)}</div>
          <div className="flex min-w-0 flex-col gap-4 lg:col-span-7">
            <SaleDetails sale={sale} />
            <div className="flex flex-wrap items-center gap-2">
              <Button variant="outline" className="w-fit" onClick={() => setIsReceiptOpen(true)}>
                <Printer aria-hidden />
                Imprimir recibo
              </Button>
              <Link className={buttonVariants({ variant: 'ghost' })} to={`/lotes/${sale.loteoId}`}>
                Ver loteo
              </Link>
            </div>
          </div>
        </div>
      )}
      <SaleReceiptDialog
        open={isReceiptOpen && sale !== null}
        receipt={sale ? saleReceiptFromSale(sale) : null}
        onClose={() => setIsReceiptOpen(false)}
      />
    </section>
  )
}
