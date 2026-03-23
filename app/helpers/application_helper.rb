module ApplicationHelper
  def public_app_host
    ENV.fetch("APP_PUBLIC_HOST", "https://linknest.info")
  end

  def public_profile_url(user_or_slug)
    slug = user_or_slug.respond_to?(:slug) ? user_or_slug.slug : user_or_slug
    "#{public_app_host}#{user_profile_path(slug)}"
  end
end
