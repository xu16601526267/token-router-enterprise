async (page) => {
  const recordedLots = []
  const refreshPayloads = []
  const adminUser = {
    id: 1,
    username: 'admin',
    display_name: 'Admin',
    role: 100,
    status: 1,
  }
  const periodStart = 1782167528
  const periodEnd = 1782174728
  const pageData = (items) => ({
    success: true,
    data: {
      page: 1,
      page_size: 100,
      total: items.length,
      items,
    },
  })
  const suppliers = [
    {
      id: 3,
      name: 'gb10-4t-self-operated',
      type: 'self_operated',
      status: 1,
      notes: 'GB10 4T self-operated prepaid supply',
      created_time: periodStart,
      updated_time: periodStart,
    },
  ]
  const capacities = [
    {
      id: 11,
      supplier_id: 3,
      supply_node: 'gb10-4t-self-operated',
      model_name: 'gpt-prepaid-process',
      period_start: periodStart,
      period_end: periodEnd,
      capacity_tokens: 1000,
      used_tokens: 320,
      headroom_tokens: 680,
      utilization_rate: 0.32,
      gpu_utilization_rate: 0.42,
      quality_score: 0.98,
      unit_cost_quota: 0.42,
      telemetry_source_type: 'node_report',
      telemetry_source_ref: 'gb10-prepaid',
      telemetry_observed_at: periodStart + 3600,
      last_telemetry_id: 5,
      status: 1,
      created_time: periodStart,
      updated_time: periodStart,
    },
  ]
  const initialLot = {
    id: 9,
    prepaid_lot_key: 'process-gb10-4t-self-operated-prepaid',
    supplier_id: 3,
    channel_id: 0,
    supply_node: 'gb10-4t-self-operated',
    model_name: 'gpt-prepaid-process',
    period_start: periodStart,
    period_end: periodEnd,
    purchased_tokens: 1000,
    unit_cost_quota: 0.42,
    total_cost_quota: 420,
    drawdown_tokens: 0,
    drawdown_request_count: 0,
    remaining_tokens: 1000,
    drawdown_rate: 0,
    drawdown_source_type: '',
    drawdown_source_ref: '',
    drawdown_refreshed_at: 0,
    source_type: 'accounting',
    source_ref: 'process-gb10-4t-self-operated-prepaid',
    observed_at: periodStart + 3600,
    external_ref: 'po://gb10-4t-self-operated',
    recorded_by: 1,
    notes: 'Process prepaid evidence',
    created_at: periodStart + 3600,
    updated_at: periodStart + 3600,
  }
  let prepaidLots = [initialLot]

  await page.unroute('**/api/**').catch(() => {})
  await page.route('**/api/**', async (route) => {
    const request = route.request()
    const path = request
      .url()
      .replace(/^https?:\/\/[^/]+/, '')
      .split('?')[0]
      .replace(/\/$/, '')
    const method = request.method()
    let body

    if (path === '/api/setup') {
      body = { success: true, data: { status: true } }
    } else if (path === '/api/status') {
      body = {
        success: true,
        data: {
          status: true,
          announcements_enabled: false,
          announcements: [],
        },
      }
    } else if (path === '/api/notice') {
      body = { success: true, data: '' }
    } else if (path === '/api/user/self') {
      body = { success: true, data: adminUser }
    } else if (path === '/api/suppliers') {
      body = pageData(suppliers)
    } else if (path === '/api/supply_capacities') {
      body = pageData(capacities)
    } else if (path === '/api/supply_prepaid_lots' && method === 'GET') {
      body = pageData(prepaidLots)
    } else if (
      path === '/api/supply_prepaid_lots/record' &&
      method === 'POST'
    ) {
      const payload = JSON.parse(request.postData() || '{}')
      recordedLots.push(payload)
      const recorded = {
        ...initialLot,
        id: 10,
        prepaid_lot_key: 'playwright-recorded-prepaid-lot',
        purchased_tokens: payload.purchased_tokens,
        unit_cost_quota: payload.unit_cost_quota,
        total_cost_quota: payload.purchased_tokens * payload.unit_cost_quota,
        remaining_tokens: payload.purchased_tokens,
        source_ref: payload.source_ref,
        external_ref: payload.external_ref || '',
        notes: payload.notes || '',
      }
      prepaidLots = [recorded, ...prepaidLots]
      body = { success: true, data: recorded }
    } else if (
      path === '/api/supply_prepaid_lots/refresh_usage' &&
      method === 'POST'
    ) {
      refreshPayloads.push(JSON.parse(request.postData() || '{}'))
      prepaidLots = prepaidLots.map((lot) => ({
        ...lot,
        drawdown_tokens: 320,
        drawdown_request_count: 2,
        remaining_tokens: lot.purchased_tokens - 320,
        drawdown_rate: 320 / lot.purchased_tokens,
        drawdown_source_type: 'usage_ledger',
        drawdown_source_ref: `usage_ledger:prepaid_lot:${lot.id}`,
        drawdown_refreshed_at: periodStart + 7200,
      }))
      body = { success: true, data: prepaidLots }
    } else if (
      path === '/api/reports/margin_summary' ||
      path === '/api/reports/quality_summary'
    ) {
      body = { success: true, data: [] }
    } else {
      body = pageData([])
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(body),
    })
  })

  await page.addInitScript((user) => {
    window.localStorage.setItem('user', JSON.stringify(user))
    window.localStorage.setItem('uid', String(user.id))
    window.localStorage.setItem('setup_status_checked', 'true')
    window.localStorage.setItem('i18nextLng', 'en')
  }, adminUser)

  await page.goto('http://127.0.0.1:4190/token-router/')
  await page.getByText('Token Router').first().waitFor()
  await page.getByRole('tab', { name: 'Prepaid Lots' }).click()
  await page.getByText('Self-operated Prepaid Lots', { exact: true }).waitFor()
  await page.getByText('gb10-4t-self-operated').first().waitFor()
  await page
    .getByText('process-gb10-4t-self-operated-prepaid')
    .first()
    .waitFor()

  await page.getByRole('button', { name: 'Record Prepaid Lot' }).click()
  const dialog = page.getByRole('dialog', { name: 'Record Prepaid Lot' })
  await dialog.waitFor()
  await dialog.getByLabel('Source Reference').fill('playwright-prepaid-lot')
  await dialog.getByLabel('Purchased Tokens').fill('2000')
  await dialog.getByLabel('Unit Cost Quota').fill('0.25')
  await dialog.getByLabel('External Reference').fill('po://playwright-prepaid')
  await dialog
    .locator('button')
    .filter({ hasText: 'Record Prepaid Lot' })
    .click()
  await page.getByText('Prepaid lot recorded').waitFor()
  await page.getByText('playwright-prepaid-lot').waitFor()

  await page.getByRole('button', { name: 'Refresh Drawdown' }).click()
  await page.getByText('Prepaid lot drawdown refreshed').waitFor()
  await page.getByText('Used: 320').first().waitFor()
  await page.getByText(/usage_ledger/).first().waitFor()

  await page.screenshot({
    path: 'output/playwright/adr0069-prepaid-lot-dashboard.png',
    fullPage: true,
  })

  return {
    url: page.url(),
    recordedLots,
    refreshPayloads,
    hasPrepaidTab:
      (await page.getByRole('tab', { name: 'Prepaid Lots' }).count()) > 0,
    hasDrawdownEvidence: (await page.getByText('Used: 320').count()) > 0,
  }
}
