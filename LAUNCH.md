# Launch playbook

## Offer

**Managed botdo workspace — $29/month**

For solo developers and small agencies using coding agents:

- one private agent task board
- live execution status and logs
- spaces for multiple repositories
- API access for agent polling
- setup help and data export

Start with dedicated single-customer deployments. This matches the product's
current isolation model and avoids pretending it is a shared multi-tenant
service. Move to shared tenancy only after demand is demonstrated.

## Zero-budget launch checklist

1. Create a payment-provider account and a recurring `$29/month` product.
2. Deploy the container on an account with persistent storage.
3. Set `BOTDO_API_KEY` to a unique random value and
   `BOTDO_CHECKOUT_URL` to the provider's checkout URL.
4. Record a 60–90 second demo: create a task, claim it through the API, and
   mark it complete while the board updates.
5. Publish the demo and offer in communities where self-promotion is allowed.
6. Contact people who have publicly asked for agent orchestration tools.
   Personalize every message; do not scrape addresses or send bulk spam.
7. On purchase, provision a unique deployment and send its URL and token.

Suggested message:

> I built a small private task board for coding agents: assign work in the
> browser, poll it over an API, and watch runs finish. I am opening a few
> managed workspaces at $29/month and will set up the first agent integration
> with you. Here is the demo: [URL]. Is this useful for your workflow?

## Funnel targets

These are hypotheses, not revenue claims:

- 30 relevant, personalized conversations
- 6 demo visits
- 2 setup calls or trials
- 1 paid workspace

One sale is `$29 MRR`, not `$29/day`. Sustained `$20–30/day` means roughly
`$600–900 MRR`, or 21–32 active workspaces at this price. Daily revenue may
also be measured as actual cash collected that day, but that is not recurring
daily income.

## Revenue evidence

Count revenue only when all of the following exist:

- a successful, non-test payment in the payment provider
- amount, currency, timestamp, and transaction identifier
- a matching active customer workspace
- refunds and provider fees recorded separately

Screenshots of a checkout page, test-mode transactions, verbal interest,
traffic, and signups are not income proof. Never publish a customer's token,
email address, full transaction identifier, or other private data.
