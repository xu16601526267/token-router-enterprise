async (page) => {
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
      id: 2,
      name: 'gb10-4t-self-hosted',
      type: 'self_hosted',
      status: 1,
      notes: 'GB10 4T self-hosted',
      created_time: periodStart,
      updated_time: periodStart,
    },
  ]
  const executions = [
    {
      id: 1,
      supply_action_plan_id: 1,
      supply_decision_id: 1,
      decision_key: 'profile:model:gpt-test|sla:default|user:2',
      traffic_profile_id: 1,
      slice_key: 'model:gpt-test|sla:default|user:2',
      model_name: 'gpt-test',
      sla_tier: 'default',
      user_id: 2,
      period_start: periodStart,
      period_end: periodEnd,
      decision_type: 'self_hosted_evaluate',
      track: 'self_hosted',
      action_type: 'evaluate_self_hosted_capacity',
      execution_status: 'recorded',
      supplier_id: 2,
      channel_id: 3,
      supply_capacity_id: 3,
      recommended_capacity: 300,
      actual_capacity_tokens: 3000,
      gap_tokens: 0,
      roi_score: 191,
      unit_cost_quota: 0.35,
      drawdown_tokens: 160,
      drawdown_request_count: 1,
      remaining_tokens: 2840,
      drawdown_rate: 0.05333333333333334,
      drawdown_source_type: 'usage_ledger',
      drawdown_source_ref: 'usage_ledger:execution:1',
      drawdown_refreshed_at: periodStart + 3601,
      effective_from: periodStart,
      effective_to: periodEnd,
      external_ref: 'process-e2e-self-hosted-routing-ready',
      operator_note: 'process e2e self-hosted routing ready',
      action_plan_completed_at: periodStart + 3600,
      action_plan_completed_by: 1,
      recorded_at: periodStart + 3600,
      recorded_by: 1,
      created_at: periodStart + 3600,
      updated_at: periodStart + 3601,
    },
  ]

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
    } else if (
      path === '/api/supply_action_executions/refresh_usage' &&
      method === 'POST'
    ) {
      refreshPayloads.push(JSON.parse(request.postData() || '{}'))
      body = { success: true, data: executions }
    } else if (path === '/api/supply_action_executions') {
      body = pageData(executions)
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
  await page.getByRole('tab', { name: 'Executions' }).click()
  await page.getByText('Supply Action Executions', { exact: true }).waitFor()
  await page.getByRole('button', { name: 'Refresh Drawdown' }).click()
  await page.getByText('Execution drawdown refreshed').first().waitFor()
  await page.getByText('Drawdown Tokens').waitFor()
  await page.getByText('Used: 160').waitFor()
  await page.getByText('Remaining: 2.8K').waitFor()
  await page.getByText(/usage_ledger/).waitFor()

  await page.screenshot({
    path: 'output/playwright/adr0062-execution-drawdown-dashboard.png',
    fullPage: true,
  })

  return {
    url: page.url(),
    refreshPayloads,
    hasDrawdownColumn:
      (await page.getByRole('columnheader', { name: 'Drawdown' }).count()) > 0,
    hasDrawdownEvidence: (await page.getByText('Remaining: 2.8K').count()) > 0,
  }
}
