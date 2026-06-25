async (page) => {
  const baseUrl = 'http://127.0.0.1:4190/token-router/'
  const desktopScreenshot =
    'output/playwright/adr0073-supplier-posture-dashboard.png'
  const mobileScreenshot =
    'output/playwright/adr0073-supplier-posture-dashboard-mobile.png'
  const mobileTableScreenshot =
    'output/playwright/adr0073-supplier-posture-dashboard-mobile-table.png'
  const periodStart = 1782162180
  const periodEnd = 1782169380
  const requests = []

  const adminUser = {
    id: 1,
    username: 'playwright-admin',
    display_name: 'Playwright Admin',
    role: 100,
    status: 1,
    group: 'default',
    quota: 1000000,
    used_quota: 0,
    request_count: 0,
  }

  const supplier = {
    id: 3,
    name: 'gb10-4t-self-operated',
    type: 'self_operated',
    status: 1,
    notes: 'GB10 4T posture supplier',
    created_time: periodStart,
    updated_time: periodStart,
  }

  const baseRecommendation = {
    id: 71,
    supplier_id: supplier.id,
    supplier_scorecard_id: 41,
    period_start: periodStart,
    period_end: periodEnd,
    score: 58,
    grade: 'D',
    recommended_action: 'disable',
    reason: 'quality and capacity incidents exceed posture threshold',
    quality_insight_count: 2,
    capacity_insight_count: 1,
    action_insight_count: 1,
    total_requests: 1200,
    success_rate: 0.82,
    avg_latency_ms: 1880,
    supply_headroom_tokens: 12000,
    avg_supply_quality_score: 0.58,
    supplier_status_current: 1,
    supplier_status_before: 0,
    supplier_status_after: 0,
    status: 'draft',
    reviewed_at: 0,
    reviewed_by: 0,
    review_note: '',
    applied_at: 0,
    applied_by: 0,
    applied_note: '',
    created_at: periodStart + 60,
    updated_at: periodStart + 60,
  }

  const approvedRecommendation = {
    ...baseRecommendation,
    id: 72,
    supplier_scorecard_id: 42,
    score: 62,
    grade: 'C',
    status: 'approved',
    reviewed_at: periodStart + 300,
    reviewed_by: 1,
    review_note: 'approved supplier posture recommendation from dashboard',
  }

  const appliedRecommendation = {
    ...approvedRecommendation,
    id: 73,
    supplier_scorecard_id: 43,
    status: 'applied',
    applied_at: periodStart + 600,
    applied_by: 1,
    applied_note: 'applied supplier posture recommendation from dashboard',
    supplier_status_before: 1,
    supplier_status_after: 2,
  }

  let postureRecommendations = [
    baseRecommendation,
    approvedRecommendation,
    appliedRecommendation,
  ]

  const ok = (data) => ({ success: true, message: '', data })
  const pageData = (items) =>
    ok({ page: 1, page_size: 100, total: items.length, items })
  const assert = (condition, message) => {
    if (!condition) throw new Error(message)
  }
  const waitForRequest = async (predicate, message) => {
    for (let i = 0; i < 50; i += 1) {
      if (requests.some(predicate)) return
      await page.waitForTimeout(100)
    }
    throw new Error(message)
  }
  const parseSearch = (search) => {
    const params = {}
    if (!search) return params
    for (const pair of search.split('&')) {
      if (!pair) continue
      const equalsIndex = pair.indexOf('=')
      const rawKey = equalsIndex === -1 ? pair : pair.slice(0, equalsIndex)
      const rawValue = equalsIndex === -1 ? '' : pair.slice(equalsIndex + 1)
      params[decodeURIComponent(rawKey)] = decodeURIComponent(rawValue)
    }
    return params
  }
  const parseRequestUrl = (rawUrl) => {
    const normalized = rawUrl.replace(/^https?:\/\/[^/]+/, '')
    const [pathname, search = ''] = normalized.split('?')
    return {
      pathname: pathname.replace(/\/+$/, ''),
      params: parseSearch(search),
      search,
    }
  }
  const fulfillJson = async (route, body) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(body),
    })
  }

  await page.context().clearCookies()
  await page.addInitScript((user) => {
    window.localStorage.setItem('user', JSON.stringify(user))
    window.localStorage.setItem('uid', String(user.id))
    window.localStorage.setItem('setup_status_checked', 'true')
    window.localStorage.setItem('i18nextLng', 'en')
    window.localStorage.setItem('language', 'en')
  }, adminUser)

  await page.route('**/api/**', async (route) => {
    const request = route.request()
    const url = parseRequestUrl(request.url())
    const path = url.pathname
    const method = request.method()
    const postData = request.postData()
    requests.push({
      method,
      path,
      search: url.search,
      params: url.params,
      postData,
    })

    if (path === '/api/user/self') {
      await fulfillJson(route, ok(adminUser))
      return
    }
    if (path === '/api/status') {
      await fulfillJson(route, ok({ version: 'playwright', footer_html: '' }))
      return
    }
    if (path === '/api/setup') {
      await fulfillJson(
        route,
        ok({ status: true, root_init: true, database_type: 'sqlite' })
      )
      return
    }
    if (path === '/api/notice') {
      await fulfillJson(route, ok(''))
      return
    }
    if (path === '/api/suppliers' && method === 'GET') {
      await fulfillJson(route, pageData([supplier]))
      return
    }
    if (
      path === '/api/supplier_posture_recommendations' &&
      method === 'GET'
    ) {
      await fulfillJson(route, pageData(postureRecommendations))
      return
    }
    if (
      path === '/api/supplier_posture_recommendations/generate' &&
      method === 'POST'
    ) {
      postureRecommendations = [baseRecommendation, approvedRecommendation]
      await fulfillJson(route, ok(postureRecommendations))
      return
    }
    if (
      path === '/api/supplier_posture_recommendations/71/approve' &&
      method === 'POST'
    ) {
      postureRecommendations = [
        {
          ...baseRecommendation,
          status: 'approved',
          reviewed_at: periodStart + 360,
          reviewed_by: 1,
          review_note:
            'approved supplier posture recommendation from dashboard',
        },
        approvedRecommendation,
      ]
      await fulfillJson(route, ok(postureRecommendations[0]))
      return
    }
    if (
      path === '/api/supplier_posture_recommendations/72/apply' &&
      method === 'POST'
    ) {
      postureRecommendations = [baseRecommendation, appliedRecommendation]
      await fulfillJson(route, ok(appliedRecommendation))
      return
    }
    if (
      path === '/api/supplier_posture_recommendations/71/reject' &&
      method === 'POST'
    ) {
      const rejected = {
        ...baseRecommendation,
        status: 'rejected',
        reviewed_at: periodStart + 420,
        reviewed_by: 1,
        review_note: 'rejected supplier posture recommendation from dashboard',
      }
      postureRecommendations = [rejected, approvedRecommendation]
      await fulfillJson(route, ok(rejected))
      return
    }
    if (
      path === '/api/reports/margin_summary' ||
      path === '/api/reports/quality_summary'
    ) {
      await fulfillJson(route, ok([]))
      return
    }

    await fulfillJson(route, pageData([]))
  })

  await page.setViewportSize({ width: 1440, height: 980 })
  await page.goto(baseUrl, { waitUntil: 'domcontentloaded' })
  await page.addStyleTag({
    content: `
      [aria-label="Open Tanstack query devtools"],
      [aria-label="Open TanStack Router Devtools"] {
        display: none !important;
      }
      [data-sonner-toaster] {
        display: none !important;
      }
    `,
  })
  await page.getByLabel('Period Start').fill('2026-06-23T05:03')
  await page.getByLabel('Period End').fill('2026-06-23T07:03')
  await page.getByRole('tab', { name: 'Posture' }).click()
  await page
    .getByText('Supplier Posture Recommendations', { exact: true })
    .waitFor()
  await page.getByText('gb10-4t-self-operated').first().waitFor()
  await page.getByText('quality and capacity incidents').first().waitFor()
  await page.getByText('Quality Insights: 2').first().waitFor()
  await page.getByText('Capacity Insights: 1').first().waitFor()
  await page.getByText('Before Enabled / After Disabled').waitFor()

  await page.getByRole('button', { name: 'Generate Posture' }).click()
  await page.getByText('Supplier posture recommendations generated').waitFor()

  await page
    .getByRole('group', { name: 'Posture Status' })
    .getByRole('button', { name: 'Applied', exact: true })
    .click()
  await page
    .getByRole('group', { name: 'Posture Action' })
    .getByRole('button', { name: 'Disable', exact: true })
    .click()

  await page
    .getByRole('group', { name: 'Posture Status' })
    .getByRole('button', { name: 'Draft', exact: true })
    .click()
  await page
    .getByRole('group', { name: 'Posture Action' })
    .getByRole('button', { name: 'All', exact: true })
    .click()

  await page
    .getByRole('row', { name: /#71/ })
    .getByRole('button', { name: 'Approve', exact: true })
    .click()
  await waitForRequest(
    (entry) =>
      entry.method === 'POST' &&
      entry.path === '/api/supplier_posture_recommendations/71/approve',
    'approve POST was not observed after clicking Approve'
  )

  await page
    .getByRole('group', { name: 'Posture Status' })
    .getByRole('button', { name: 'Approved', exact: true })
    .click()
  await page
    .getByRole('row', { name: /#72/ })
    .getByRole('button', { name: 'Apply', exact: true })
    .click()
  await waitForRequest(
    (entry) =>
      entry.method === 'POST' &&
      entry.path === '/api/supplier_posture_recommendations/72/apply',
    'apply POST was not observed after clicking Apply'
  )
  await page.getByText('Before Enabled / After Disabled').waitFor()

  await page
    .getByRole('group', { name: 'Posture Status' })
    .getByRole('button', { name: 'Draft', exact: true })
    .click()
  await page
    .getByRole('row', { name: /#71/ })
    .getByRole('button', { name: 'Reject', exact: true })
    .click()
  await waitForRequest(
    (entry) =>
      entry.method === 'POST' &&
      entry.path === '/api/supplier_posture_recommendations/71/reject',
    'reject POST was not observed after clicking Reject'
  )

  await page.evaluate(() => {
    for (const element of document.querySelectorAll('*')) {
      element.scrollLeft = 0
    }
  })
  await page.screenshot({ path: desktopScreenshot, fullPage: true })

  const postureGets = requests.filter(
    (entry) =>
      entry.method === 'GET' &&
      entry.path === '/api/supplier_posture_recommendations'
  )
  const generatePosts = requests.filter(
    (entry) =>
      entry.method === 'POST' &&
      entry.path === '/api/supplier_posture_recommendations/generate'
  )
  const approvePosts = requests.filter(
    (entry) =>
      entry.method === 'POST' &&
      entry.path === '/api/supplier_posture_recommendations/71/approve'
  )
  const rejectPosts = requests.filter(
    (entry) =>
      entry.method === 'POST' &&
      entry.path === '/api/supplier_posture_recommendations/71/reject'
  )
  const applyPosts = requests.filter(
    (entry) =>
      entry.method === 'POST' &&
      entry.path === '/api/supplier_posture_recommendations/72/apply'
  )

  assert(
    postureGets.some((entry) => entry.params.status === 'draft'),
    'default draft posture GET was not observed'
  )
  assert(
    postureGets.some(
      (entry) =>
        entry.params.status === 'applied' &&
        entry.params.recommended_action === 'disable'
    ),
    'applied disable posture filter GET was not observed'
  )
  assert(generatePosts.length === 1, 'generate POST was not observed once')
  assert(approvePosts.length === 1, 'approve POST was not observed once')
  assert(rejectPosts.length === 1, 'reject POST was not observed once')
  assert(applyPosts.length === 1, 'apply POST was not observed once')

  const generatePayload = JSON.parse(generatePosts[0].postData || '{}')
  assert(
    generatePayload.period_start === periodStart &&
      generatePayload.period_end === periodEnd,
    `generate POST did not include the dashboard period: ${JSON.stringify(
      generatePayload
    )}`
  )
  assert(
    JSON.parse(approvePosts[0].postData || '{}').review_note.includes(
      'approved supplier posture'
    ),
    'approve POST review_note drifted'
  )
  assert(
    JSON.parse(rejectPosts[0].postData || '{}').review_note.includes(
      'rejected supplier posture'
    ),
    'reject POST review_note drifted'
  )
  assert(
    JSON.parse(applyPosts[0].postData || '{}').operator_note.includes(
      'applied supplier posture'
    ),
    'apply POST operator_note drifted'
  )

  await page.setViewportSize({ width: 390, height: 844 })
  await page
    .getByText('Supplier Posture Recommendations', { exact: true })
    .scrollIntoViewIfNeeded()
  await page.waitForTimeout(300)
  await page.screenshot({ path: mobileScreenshot, fullPage: false })
  await page.getByText('Quality Insights: 2').first().scrollIntoViewIfNeeded()
  await page.waitForTimeout(300)
  await page.screenshot({ path: mobileTableScreenshot, fullPage: false })

  console.log(
    JSON.stringify({
      verified: true,
      screenshots: [desktopScreenshot, mobileScreenshot, mobileTableScreenshot],
      postureGetCount: postureGets.length,
      generatePostCount: generatePosts.length,
      approvePostCount: approvePosts.length,
      rejectPostCount: rejectPosts.length,
      applyPostCount: applyPosts.length,
    })
  )
}
