import { Controller } from "@hotwired/stimulus"

export default class extends Controller {
  async trackClick(event) {
    const linkId = event.target.closest('a').dataset.linkId
    if (!linkId) return

    try {
      // Send tracking request asynchronously (don't block the redirect)
      await fetch(`/links/${linkId}/track_click`, {
        method: 'POST',
        headers: {
          'X-CSRF-Token': document.querySelector('meta[name="csrf-token"]').content
        }
      }).catch(() => {
        // Silently ignore errors - click tracking shouldn't block navigation
        console.debug('Click tracking request sent')
      })
    } catch (error) {
      console.debug('Click tracking error:', error)
    }

    // Allow the link to navigate normally
    return true
  }
}
