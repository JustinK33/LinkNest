module ApplicationHelper
  def public_app_host
    ENV.fetch("APP_PUBLIC_HOST", "https://linknest.info")
  end

  def safe_link_href(raw_url, fallback = "#")
    return fallback if raw_url.blank?

    url = raw_url.to_s.strip
    return fallback if url.blank?
    return url if url.start_with?("/")

    parsed = URI.parse(url)
    return url if %w[http https mailto tel].include?(parsed.scheme)

    fallback
  rescue URI::InvalidURIError
    fallback
  end

  def public_profile_url(user_or_slug)
    slug = user_or_slug.respond_to?(:slug) ? user_or_slug.slug : user_or_slug
    "#{public_app_host}#{user_profile_path(slug)}"
  end
end
