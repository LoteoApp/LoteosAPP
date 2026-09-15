import { useState } from 'react'
import { Link } from 'react-router'
import { Alert, AlertDescription } from '../../../shared/ui/alert'
import { Button } from '../../../shared/ui/button'
import SaleFilters from '../components/SaleFilters'
import SalesList from '../components/SalesList'
import SalesListSkeleton from '../components/SalesListSkeleton'
import SalesPagination from '../components/SalesPagination'
import { useSales } from '../hooks/use-sales'
import type { SaleState } from '../types'

type SalesPageProps = { accessToken?: string }

export default function SalesPage({ accessToken = '' }: SalesPageProps) {
  const [search, setSearch] = useState('')
  const [state, setState] = useState<SaleState | ''>('')
  const [pageNumber, setPageNumber] = useState(1)
  const sales = useSales(accessToken, { q: search || undefined, estado: state || undefined, pagina: pageNumber })

  return (
    <section className="flex flex-col gap-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-foreground">Ventas</h1>
          <p className="text-sm text-muted-foreground">
            Consultá las ventas registradas desde el visor de lotes.
          </p>
        </div>
        <Button render={<Link to="/lotes" />}>Abrir visor de lotes</Button>
      </div>
      <section className="grid gap-3">
        <SaleFilters
          search={search}
          state={state}
          onSearchChange={(value) => {
            setSearch(value)
            setPageNumber(1)
          }}
          onStateChange={(value) => {
            setState(value)
            setPageNumber(1)
          }}
        />
        {sales.error && (
          <Alert variant="destructive">
            <AlertDescription>{sales.error}</AlertDescription>
          </Alert>
        )}
        {sales.isLoading ? <SalesListSkeleton /> : <SalesList sales={sales.page.ventas} />}
        <SalesPagination page={sales.page} isLoading={sales.isLoading} onPageChange={setPageNumber} />
      </section>
    </section>
  )
}
