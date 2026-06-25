async (page) => {
  const recordedCostProfiles = []
  const adminUser = {
    id: 1,
    username: 'admin',
    display_name: 'Admin',
    role: 100,
    status: 1,
  }
  const periodStart = 1782144000
  const periodEnd = 1782230400
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
      notes: 'GB10 4T',
      created_time: periodStart,
      updated_time: periodStart,
    },
  ]
  const costProfiles = [
    {
      id: 7,
      cost_profile_key: 'gb10-4t-cost-20260622',
      supplier_id: 2,
      supply_node: 'gb10-4t-self-hosted',
      model_name: 'qwen3-32b',
      period_start: periodStart,
      period_end: periodEnd,
      capacity_tokens: 1000,
      fixed_cost_quota: 100,
      variable_unit_cost_quota: 0.02,
      amortized_unit_cost_quota: 0.12,
      source_type: 'accounting',
      source_ref: 'process-gb10-4t-self-hosted-cost',
      observed_at: periodStart + 3600,
      recorded_by: 1,
      notes: 'Process evidence',
      created_at: periodStart + 3600,
      updated_at: periodStart + 3600,
    },
  ]
  const capacities = [
    {
      id: 11,
      supplier_id: 2,
      supply_node: 'gb10-4t-self-hosted',
      model_name: 'qwen3-32b',
      period_start: periodStart,
      period_end: periodEnd,
      capacity_tokens: 1000,
      used_tokens: 700,
      headroom_tokens: 300,
      utilization_rate: 0.7,
      gpu_utilization_rate: 0.68,
      quality_score: 0.99,
      unit_cost_quota: 0.12,
      telemetry_source_type: 'node_report',
      telemetry_source_ref: 'gb10-process',
      telemetry_observed_at: periodStart + 3600,
      last_telemetry_id: 5,
      status: 1,
      created_time: periodStart,
      updated_time: periodStart,
    },
  ]
  const opportunities = [
    {
      id: 31,
      opportunity_key: 'opp-self-hosted-gb10',
      supply_decision_id: 21,
      decision_source: 'profile',
      traffic_profile_id: 14,
      traffic_forecast_id: 0,
      forecast_target_period_start: 0,
      forecast_target_period_end: 0,
      forecast_confidence: 0,
      forecast_method: '',
      model_name: 'qwen3-32b',
      sla_tier: 'standard',
      user_id: 1,
      period_start: periodStart,
      period_end: periodEnd,
      track: 'self_hosted',
      decision_type: 'self_hosted_evaluate',
      decision_status: 'approved',
      opportunity_type: 'self_hosted_cache',
      priority: 'action',
      cluster_key: 'high_cache_stable',
      demand_tokens: 900,
      supply_headroom_tokens: 300,
      gap_tokens: 600,
      recommended_capacity: 300,
      avg_unit_cost_quota: 0.5,
      avg_supply_quality_score: 0.99,
      cache_hit_rate: 0.42,
      locality_score: 0.9,
      stability_score: 0.8,
      headroom_risk_score: 0.6,
      self_hosted_cost_profile_id: 7,
      self_hosted_unit_cost_quota: 0.12,
      self_hosted_savings_unit_quota: 0.38,
      self_hosted_savings_quota: 114,
      rank_score: 405,
      reason: 'GB10 cost basis supports self-hosted cache expansion.',
      generated_at: periodStart + 7200,
      created_at: periodStart + 7200,
      updated_at: periodStart + 7200,
    },
  ]

  await page.route('**/api/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    const path = url.pathname.replace(/\/$/, '')
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
    } else if (path === '/api/supply_cost_profiles' && method === 'GET') {
      body = pageData(costProfiles)
    } else if (
      path === '/api/supply_cost_profiles/record' &&
      method === 'POST'
    ) {
      const payload = JSON.parse(request.postData() || '{}')
      recordedCostProfiles.push(payload)
      const recorded = {
        ...costProfiles[0],
        id: 8,
        cost_profile_key: 'playwright-recorded-cost',
        fixed_cost_quota: payload.fixed_cost_quota,
        variable_unit_cost_quota: payload.variable_unit_cost_quota,
        source_ref: payload.source_ref,
        notes: payload.notes || '',
      }
      body = { success: true, data: recorded }
    } else if (path === '/api/supply_expansion_opportunities') {
      body = pageData(opportunities)
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

  await page.getByRole('tab', { name: 'Cost Profiles' }).click()
  await page.getByText('Self-hosted Cost Profiles', { exact: true }).waitFor()
  await page.getByText('gb10-4t-self-hosted').first().waitFor()
  await page.getByText('process-gb10-4t-self-hosted-cost').waitFor()

  await page.getByRole('tab', { name: 'Opportunities' }).click()
  await page.getByText('Cost Profile #7').waitFor()
  await page.getByText(/Total Savings:.*0\.000228/).waitFor()
  const opportunityEvidenceVerified =
    (await page.getByText('Cost Profile #7').count()) > 0

  await page.getByRole('tab', { name: 'Cost Profiles' }).click()
  await page.getByRole('button', { name: 'Record Cost Profile' }).click()
  await page.getByRole('dialog', { name: 'Record Cost Profile' }).waitFor()
  await page.getByLabel('Source Reference').fill('playwright-cost-profile')
  await page.getByLabel('Fixed Cost Quota').fill('120')
  await page.getByLabel('Variable Unit Cost').fill('0.02')
  await page.locator('[role="dialog"] button').filter({
    hasText: 'Record Cost Profile',
  }).click()
  await page.getByText('Cost profile recorded').waitFor()

  await page.screenshot({
    path: 'output/playwright/adr0061-cost-profile-dashboard.png',
    fullPage: true,
  })

  return {
    url: page.url(),
    postedProfiles: recordedCostProfiles,
    hasCostProfileTab: (await page.getByRole('tab', {
      name: 'Cost Profiles',
    }).count()) > 0,
    hasOpportunityEvidence: opportunityEvidenceVerified,
  }
}
