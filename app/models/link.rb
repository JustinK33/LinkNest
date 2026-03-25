class Link < ApplicationRecord
  belongs_to :user
  has_many :link_clicks, dependent: :destroy
  has_many :hourly_link_stats, dependent: :destroy
  has_one_attached :resume_pdf

  validates :title, presence: true
  validate :url_or_resume_present
  validate :url_format_is_valid
  validate :resume_pdf_must_be_pdf
  validate :resume_pdf_size_limit
  validate :resume_pdf_content_validation
  validates :user_id, presence: true
  validates :position, presence: true, numericality: { only_integer: true }

  scope :ordered, -> { order(:position) }
  scope :active, -> { where(deleted_at: nil) }
  scope :public_links, -> {
    if column_names.include?('public')
      where(public: true)
    else
      all # Return all links if public column doesn't exist
    end
  }
  scope :private_links, -> {
    if column_names.include?('public')
      where(public: false)
    else
      none # Return no links if public column doesn't exist
    end
  }

  # Update click count from aggregated stats
  def update_click_count!
    total_clicks = link_clicks.count
    update(click_count: total_clicks)
  end

  # Safe method to check if link is public (handles missing column)
  def public?
    if self.class.column_names.include?('public')
      read_attribute(:public) != false # Default to true if nil
    else
      true # Default to public if column doesn't exist
    end
  end

  # Get clicks for a time period
  def clicks_in_range(start_time, end_time)
    link_clicks.where(created_at: start_time..end_time)
  end

  # Get unique visitors for a time period
  def unique_visitors_in_range(start_time, end_time)
    link_clicks.where(created_at: start_time..end_time).distinct.count(:ip_address)
  end

  private
    def url_or_resume_present
      return if url.present? || resume_pdf.attached?

      errors.add(:base, "Provide a URL or upload a PDF resume")
    end

    def url_format_is_valid
      return if url.blank?
      return if url.start_with?("/")
      return if url.start_with?("mailto:", "tel:")
      return if url.match?(URI::DEFAULT_PARSER.make_regexp)

      errors.add(:url, "must be a valid URL")
    end

    def resume_pdf_must_be_pdf
      return unless resume_pdf.attached?

      # Check content type (basic check)
      unless resume_pdf.content_type == "application/pdf"
        errors.add(:resume_pdf, "must be a PDF file")
        return
      end

      # Server-side file content validation - check PDF magic bytes
      resume_pdf.blob.open do |file|
        header = file.read(4)
        unless header == "%PDF"
          errors.add(:resume_pdf, "file does not appear to be a valid PDF")
          return
        end
      end
    rescue => e
      errors.add(:resume_pdf, "could not validate file content")
    end

    def resume_pdf_size_limit
      return unless resume_pdf.attached?

      # Limit PDF files to 10MB
      max_size = 10.megabytes
      if resume_pdf.byte_size > max_size
        errors.add(:resume_pdf, "must be 10MB or smaller")
      end
    end

    def resume_pdf_content_validation
      return unless resume_pdf.attached?

      # Additional security checks
      resume_pdf.blob.open do |file|
        content = file.read(1024) # Read first 1KB for analysis

        # Check for suspicious patterns (basic malware detection)
        suspicious_patterns = [
          /javascript/i,
          /<script/i,
          /eval\(/i,
          /onload=/i,
          /iframe/i
        ]

        suspicious_patterns.each do |pattern|
          if content.match?(pattern)
            errors.add(:resume_pdf, "file contains potentially malicious content")
            break
          end
        end
      end
    rescue => e
      # Log error but don't fail validation to avoid DOS attacks
      Rails.logger.warn "Error validating PDF content: #{e.message}"
    end
end
