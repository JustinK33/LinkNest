class SlugConstraint
  RESERVED_SLUGS = %w[
    admin
    api
    dashboard
    settings
    profile
    links
    users
    sessions
    session
    passwords
    password
    pages
    pages_controller
    static
    health
    active_storage
    action_text
    rails
    up
  ].freeze

  def matches?(request)
    slug = request.path_parameters[:slug]
    return false if slug.blank?
    return false if RESERVED_SLUGS.include?(slug.downcase)

    # Slug must exist and belong to a user
    User.exists?(slug: slug)
  end
end
